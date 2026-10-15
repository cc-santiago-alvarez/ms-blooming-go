package shared_domain

// NoticeType es el typeId del aviso.
type NoticeType string

const (
	NoticeTypeBalance NoticeType = "1"
	NoticeTypeBond    NoticeType = "3"
)

// Notice son los datos de un aviso para un usuario. Title y Message ya vienen formateados.
type Notice struct {
	Type     NoticeType
	UserID   string
	AgencyID string
	Title    string
	Message  string
}
