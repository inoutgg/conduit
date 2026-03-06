package hashsum

type Store interface {
	Compare(string, []byte) (bool, []byte, error)
	Save(string, []byte) error
}
