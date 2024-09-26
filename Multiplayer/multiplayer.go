package Multiplayer

type (
	MP struct {
		IP    string
		State int
	}
)

func NewMP() *MP {
	return &MP{}
}

func Ping(ip string) bool {
	return false
}

func (mp *MP) Connect() error {
	return nil
}
