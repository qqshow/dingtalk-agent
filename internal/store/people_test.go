package store

import "testing"

func TestSearchPeople(t *testing.T) {
	s := &Snapshot{MyUID: 1, Users: map[int64]User{
		1:  {UID: 1, RealName: "王我"},               // me: excluded
		2:  {UID: 2, RealName: "王小明"},              // single chat
		3:  {UID: 3, RealName: "王芳"},               // group-only contact
		4:  {UID: 4, RealName: "考勤王", UserType: 2}, // bot: excluded
		5:  {UID: 5, Nick: "老王"},                   // matched by nickname
		6:  {UID: 6, RealName: "李雷"},               // no match
		7:  {UID: 7, RealName: "小王"},               // contains, not prefix
		8:  {UID: 8},                               // no name: excluded
		20: {UID: 20, RealName: "王五", Nick: "w5"},  // prefix match
	}}
	s.Convs = []*Conversation{{CID: "1:2", Type: "single", OtherUID: "2", LastModify: 100}}

	got := s.SearchPeople("王", 10)
	var names []string
	for _, p := range got {
		names = append(names, p.Name)
	}
	if len(got) == 0 || got[0].UID != 2 || got[0].CID != "1:2" {
		t.Fatalf("people with a single chat should come first: %v", names)
	}
	seen := map[int64]bool{}
	for _, p := range got {
		seen[p.UID] = true
		if p.UID != 2 && p.CID != "" {
			t.Errorf("%s should have no single chat", p.Name)
		}
	}
	for _, uid := range []int64{1, 4, 6, 8} {
		if seen[uid] {
			t.Errorf("uid %d should be excluded: %v", uid, names)
		}
	}
	for _, uid := range []int64{3, 5, 7, 20} {
		if !seen[uid] {
			t.Errorf("uid %d missing: %v", uid, names)
		}
	}
	// Prefix matches before "contains" matches.
	pos := map[int64]int{}
	for i, p := range got {
		pos[p.UID] = i
	}
	if pos[3] > pos[7] || pos[20] > pos[7] {
		t.Errorf("prefix matches should rank above contains matches: %v", names)
	}
	if s.SingleCIDFor(2) != "1:2" || s.SingleCIDFor(3) != "" {
		t.Error("SingleCIDFor")
	}
	if n := len(s.SearchPeople("", 3)); n != 3 {
		t.Errorf("limit: got %d", n)
	}
}
