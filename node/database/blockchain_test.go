package database

import (
	"os"
	"testing"

	"github.com/gelembjuk/oursql/lib/utils"
	assert "github.com/stretchr/testify/require"
)

const testFolderName = "testdata"

func getTestDBManagerInited() (*BoltDBManager, error) {
	obj, err := getTestDBManager()

	if err != nil {
		return nil, err
	}

	err = obj.InitDatabase()

	if err != nil {
		return nil, err
	}

	err = obj.OpenConnection("testing")

	if err != nil {
		return nil, err
	}

	return obj, nil
}
func getTestDBManager() (*BoltDBManager, error) {
	destroyTestDB(nil)

	err := os.Mkdir(testFolderName, 0744)

	if err != nil {
		return nil, err
	}

	logger := utils.CreateLogger()
	logger.EnableLogs("")
	logger.LogToStdout()

	c := DatabaseConfig{}
	c.SetDefault()
	c.ConfigDir = testFolderName + "/"

	obj := &BoltDBManager{}
	obj.SetLockerObject(obj.GetLockerObject())
	obj.SetLogger(logger)
	obj.SetConfig(c)

	return obj, nil
}

func destroyTestDB(man *BoltDBManager) {

	if man != nil {
		man.CloseConnection()
	}

	if _, err := os.Stat(testFolderName); err == nil {
		os.RemoveAll(testFolderName)
	}
}

func TestBlockChainAdd(t *testing.T) {
	man, err := getTestDBManagerInited()

	defer destroyTestDB(man)

	assert.NoError(t, err, "Can not prepare data")

	bcm, err := man.GetBlockchainObject()

	assert.NoError(t, err, "Can not get BC object")

	hash1 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	hash2 := []byte{0, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	hash3 := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	err = bcm.AddToChain(hash1, nil)

	assert.NoError(t, err, "Adding hash1")

	exists, err := bcm.BlockInChain(hash1)

	assert.NoError(t, err, "Check hash1")
	assert.True(t, exists, "Block should exist for hash1")

	err = bcm.AddToChain(hash2, hash1)

	assert.NoError(t, err, "Can not add hash2")

	// get state of hash1

	exists, prevHash, nextHash, err := bcm.GetLocationInChain(hash1)

	assert.NoError(t, err, "Check if in chain hash1 (2)")
	assert.True(t, exists, "Block should exist fir hash1 (2)")
	assert.True(t, len(prevHash) == 0, "Prev hash should not be present for hash1")
	assert.True(t, len(nextHash) > 0, "Next hash should be present hash1")

	// get state of hash2
	exists, prevHash, nextHash, err = bcm.GetLocationInChain(hash2)

	assert.NoError(t, err, "Check if in chain hash2 (2)")
	assert.True(t, exists, "Block should exist fir hash2 (2)")
	assert.True(t, len(prevHash) > 0, "Prev hash should be present for hash2")
	assert.True(t, len(nextHash) == 0, "Next hash should not be present hash2")

	err = bcm.AddToChain(hash3, hash2)

	assert.NoError(t, err, "Can not add hash3")

	// check state of hash 2 now
	exists, prevHash, nextHash, err = bcm.GetLocationInChain(hash2)

	assert.NoError(t, err, "Check if in chain hash2 (3)")
	assert.True(t, exists, "Block should exist fir hash2 (3)")
	assert.True(t, len(prevHash) > 0, "Prev hash should be present for hash2 (3)")
	assert.True(t, len(nextHash) > 0, "Next hash should not be present hash2 (3)")

	// check state of hash 3
	exists, prevHash, nextHash, err = bcm.GetLocationInChain(hash3)

	assert.NoError(t, err, "Check if in chain hash3")
	assert.True(t, exists, "Block should exist for hash3 ")
	assert.True(t, len(prevHash) > 0, "Prev hash should be present for hash3")
	assert.True(t, len(nextHash) == 0, "Next hash should not be present hash3")

	// try to add unexistent previous
	err = bcm.AddToChain(hash2, []byte{1, 2, 3})

	assert.Error(t, err, "Error should be on adding over not existent previous")

	// try to add over a hash taht already has next
	err = bcm.AddToChain(hash1, hash2)

	assert.Error(t, err, "Error should be on adding over hash with existent next hash")

}

func TestBlockChainRemove(t *testing.T) {
	man, err := getTestDBManagerInited()

	defer destroyTestDB(man)

	assert.NoError(t, err, "Can not prepare data")

	bcm, err := man.GetBlockchainObject()

	assert.NoError(t, err, "Can not get BC object")

	hash1 := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}
	hash2 := []byte{0, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	hash3 := []byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1}

	bcm.AddToChain(hash1, nil)
	bcm.AddToChain(hash2, hash1)
	bcm.AddToChain(hash3, hash2)

	// check state of hash 1
	exists, prevHash, nextHash, err := bcm.GetLocationInChain(hash1)

	assert.NoError(t, err, "Check if in chain hash1 (2)")
	assert.True(t, exists, "Block should exist fir hash1 (2)")
	assert.True(t, len(prevHash) == 0, "Prev hash should not be present for hash1")
	assert.True(t, len(nextHash) > 0, "Next hash should be present hash1")

	// check state of hash 2
	exists, prevHash, nextHash, err = bcm.GetLocationInChain(hash2)

	assert.NoError(t, err, "Check if in chain hash2 (3)")
	assert.True(t, exists, "Block should exist fir hash2 (3)")
	assert.True(t, len(prevHash) > 0, "Prev hash should be present for hash2 (3)")
	assert.True(t, len(nextHash) > 0, "Next hash should not be present hash2 (3)")

	// check state of hash 3
	exists, prevHash, nextHash, err = bcm.GetLocationInChain(hash3)

	assert.NoError(t, err, "Check if in chain hash3")
	assert.True(t, exists, "Block should exist for hash3 ")
	assert.True(t, len(prevHash) > 0, "Prev hash should be present for hash3")
	assert.True(t, len(nextHash) == 0, "Next hash should not be present hash3")

	err = bcm.RemoveFromChain(hash1)
	assert.Error(t, err, "First hash should not be able to be removed")

	err = bcm.RemoveFromChain(hash3)
	assert.NoError(t, err, "Last hash must be possible to remove")

	exists, prevHash, nextHash, err = bcm.GetLocationInChain(hash3)
	assert.NoError(t, err, "Error loading non existent hash")

	assert.False(t, exists, "Hash3 should not be in chain")

	// hash2 now becomes last one
	exists, prevHash, nextHash, err = bcm.GetLocationInChain(hash2)

	assert.NoError(t, err, "Check if in chain hash2 (4)")
	assert.True(t, exists, "Block should exist fir hash2 (4)")
	assert.True(t, len(prevHash) > 0, "Prev hash should be present for hash2 (3)")
	assert.True(t, len(nextHash) == 0, "No next hash for hash2")
}
