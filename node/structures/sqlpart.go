package structures

// SQL Transaction keeps a query and rollback query to cancel this update
type SQLUpdate struct {
	ReferenceID     []byte
	Query           []byte
	RollbackQuery   []byte
	PrevTransaction []byte
}

func (q SQLUpdate) IsEmpty() bool {
	if len(q.Query) == 0 {
		return true
	}
	return false
}

func (q SQLUpdate) ToBytes() []byte {
	bs := q.ReferenceID[:]
	bs = append(bs, q.Query[:]...)
	bs = append(bs, q.RollbackQuery[:]...)
	return bs
}
