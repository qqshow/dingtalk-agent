package agentdb

import (
	"path/filepath"
	"testing"
)

func TestChatSessions(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s, err := db.CreateChatSession("项目进展")
	if err != nil {
		t.Fatal(err)
	}
	u, err := db.AddChatMessage(ChatMessage{SessionID: s.ID, Role: "user", Content: "@项目群 最近怎么样",
		Context: "### @项目群\n10:00 张三: 上线了", Meta: ChatMeta{Mentions: []Mention{{CID: "g1", Title: "项目群", Kind: "group"}}, Days: 7}})
	if err != nil {
		t.Fatal(err)
	}
	db.AddChatMessage(ChatMessage{SessionID: s.ID, Role: "assistant", Content: "已上线",
		Meta: ChatMeta{Model: "m", Tools: []ToolUse{{Name: "dingtalk_get_messages", Title: "读取消息"}}}})

	msgs, err := db.ChatMessages(s.ID)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("messages = %+v, %v", msgs, err)
	}
	if msgs[0].ID != u.ID || msgs[0].Context == "" || msgs[0].Meta.Mentions[0].CID != "g1" || msgs[1].Meta.Tools[0].Title != "读取消息" {
		t.Fatalf("round trip = %+v", msgs)
	}
	list, _ := db.ListChatSessions(10)
	if len(list) != 1 || list[0].Messages != 2 {
		t.Fatalf("sessions = %+v", list)
	}
	if err := db.RenameChatSession(s.ID, "新标题"); err != nil {
		t.Fatal(err)
	}
	if got, _ := db.GetChatSession(s.ID); got.Title != "新标题" {
		t.Fatalf("title = %q", got.Title)
	}
	if err := db.DeleteChatSession(s.ID); err != nil {
		t.Fatal(err)
	}
	if msgs, _ := db.ChatMessages(s.ID); len(msgs) != 0 {
		t.Fatal("messages not deleted with session")
	}
	if _, err := db.GetChatSession(s.ID); err == nil {
		t.Fatal("session still exists")
	}
}
