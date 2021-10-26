package records

import "sync"

type Record struct {
	IPs []string
	TTL int
}

type RecordsList struct {
	sync.RWMutex
	Records map[string]Record
}

func (rl *RecordsList) Append(dn string, addr []string, TTL int) {
	r := Record{addr, TTL}
	rl.Lock()
	rl.Records[dn] = r
	rl.Unlock()
}

func (rl *RecordsList) Read(dn string, t int) (bool, []string) {
	rl.Lock()
	r, ok := rl.Records[dn]
	rl.Unlock()
	if !ok || t > r.TTL {
		return false, nil
	} else {
		return true, r.IPs
	}
}

func NewRecordsList() *RecordsList {
	var rl RecordsList
	rl.Records = make(map[string]Record)
	return &rl
}
