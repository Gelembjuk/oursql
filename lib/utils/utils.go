package utils

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"

	"golang.org/x/crypto/ripemd160"

	"github.com/gelembjuk/oursql/lib"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Structure to manage logs
type LoggerMan struct {
	State   map[string]bool
	loggers map[string]*os.File
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
}

// Creates logger object. sets all logging to STDOUT
func CreateLogger() *LoggerMan {
	logger := LoggerMan{}
	logger.loggers = map[string]*os.File{"trace": nil, "error": nil, "info": nil, "warning": nil}

	logger.State = map[string]bool{"trace": false, "error": false, "info": false, "warning": false}

	logger.Trace = log.New(ioutil.Discard,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	logger.Info = log.New(ioutil.Discard,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	logger.Warning = log.New(ioutil.Discard,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	logger.Error = log.New(ioutil.Discard,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	return &logger
}
func CreateLoggerStdout() *LoggerMan {
	logger := CreateLogger()
	logger.LogToStdout()
	return logger
}

// change enabled logs state
func (logger *LoggerMan) EnableLogs(logs string) {
	l := strings.Split(logs, ",")

	for _, lv := range l {
		logger.State[lv] = true
	}
}

// return logs state as command separated list
func (logger *LoggerMan) GetState() string {
	list := []string{}

	for l, state := range logger.State {
		if state {
			list = append(list, l)
		}
	}

	return strings.Join(list, ",")
}

// disable all logging
func (logger *LoggerMan) DisableLogging() {
	logger.Trace.SetOutput(ioutil.Discard)
	logger.Info.SetOutput(ioutil.Discard)
	logger.Warning.SetOutput(ioutil.Discard)
	logger.Error.SetOutput(ioutil.Discard)

	for t, p := range logger.loggers {
		if p != nil {
			p.Close()
		}
		logger.loggers[t] = nil
	}
}

// Changes logging to files
func (logger *LoggerMan) LogToFiles(datadir, trace, info, warning, errorname string) error {
	if logger.State["trace"] {
		f1, err1 := os.OpenFile(datadir+trace, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

		if err1 == nil {
			logger.loggers["trace"] = f1
			logger.Trace.SetOutput(f1)
		}
	}
	if logger.State["info"] {
		f2, err2 := os.OpenFile(datadir+info, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

		if err2 == nil {
			logger.loggers["info"] = f2
			logger.Info.SetOutput(f2)
		}
	}
	if logger.State["warning"] {
		f3, err3 := os.OpenFile(datadir+warning, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

		if err3 == nil {
			logger.loggers["warning"] = f3
			logger.Warning.SetOutput(f3)
		}
	}
	if logger.State["error"] {
		f4, err4 := os.OpenFile(datadir+errorname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

		if err4 == nil {
			logger.loggers["error"] = f4
			logger.Error.SetOutput(f4)
		}
	}
	return nil
}

// Sets ogging to STDOUT
func (logger *LoggerMan) LogToStdout() error {
	if logger.State["trace"] {
		logger.Trace.SetOutput(os.Stdout)
	}
	if logger.State["info"] {
		logger.Info.SetOutput(os.Stdout)
	}
	if logger.State["warning"] {
		logger.Warning.SetOutput(os.Stdout)
	}
	if logger.State["error"] {
		logger.Error.SetOutput(os.Stdout)
	}
	return nil
}

// IntToHex converts an int64 to a byte array
func IntToHex(num int64) []byte {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

// ReverseBytes reverses a byte array
func ReverseBytes(data []byte) {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
}

// Converts address string to hash of pubkey
func AddresToPubKeyHash(address string) ([]byte, error) {
	pubKeyHash := Base58Decode([]byte(address))

	if len(pubKeyHash) < 10 {
		return nil, errors.New("Wrong address")
	}

	pubKeyHash = pubKeyHash[1 : len(pubKeyHash)-4]

	return pubKeyHash, nil
}

// Converts hash of pubkey to address as a string
func PubKeyHashToAddres(pubKeyHash []byte) (string, error) {
	versionedPayload := append([]byte{lib.Version}, pubKeyHash...)

	checksum := Checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)

	return fmt.Sprintf("%s", address), nil
}

// Makes string adres from pub key
func PubKeyToAddres(pubKey []byte) (string, error) {
	pubKeyHash, err := HashPubKey(pubKey)

	if err != nil {
		return "", err
	}
	versionedPayload := append([]byte{lib.Version}, pubKeyHash...)

	checksum := Checksum(versionedPayload)

	fullPayload := append(versionedPayload, checksum...)
	address := Base58Encode(fullPayload)

	return fmt.Sprintf("%s", address), nil
}

// Checksum generates a checksum for a public key
func Checksum(payload []byte) []byte {
	firstSHA := sha256.Sum256(payload)
	secondSHA := sha256.Sum256(firstSHA[:])

	return secondSHA[:lib.AddressChecksumLen]
}

// HashPubKey hashes public key
func HashPubKey(pubKey []byte) ([]byte, error) {
	publicSHA256 := sha256.Sum256(pubKey)

	RIPEMD160Hasher := ripemd160.New()
	_, err := RIPEMD160Hasher.Write(publicSHA256[:])
	if err != nil {
		return nil, err
	}
	publicRIPEMD160 := RIPEMD160Hasher.Sum(nil)

	return publicRIPEMD160, nil
}

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func CopyBytes(source []byte) []byte {
	if len(source) > 0 {
		d := make([]byte, len(source))

		copy(d, source)

		return d
	}
	return []byte{}
}
