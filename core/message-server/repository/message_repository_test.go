package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/kageos/kageos/core/message-server/model"
	"github.com/kageos/kageos/dto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListInboxScansMessageInboxDTO(t *testing.T) {
	repo := newTestMessageRepo(t)

	_, err := repo.Create(context.Background(), dto.MessageSendMeta{
		From:               "alice",
		FullCodePath:       "/alice/hr/leave.form",
		SourceType:         "scheduled_task",
		SourcePath:         "/alice/hr/leave.form",
		SourceTitle:        "请假审批",
		SourceParentPath:   "/alice/hr",
		SourceParentTitle:  "人事系统",
		SourceTemplateType: "form",
		SourceRef:          "timer_task:12:execution:34",
		WorkspaceSessionID: "session-1",
	}, dto.MessageSendPayload{
		Title:   "请假审批",
		Content: "请审批",
		Files:   "kageos/reports/leave.pdf",
	}, []string{"bob"})
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	list, total, err := repo.ListInbox(context.Background(), "bob", InboxListFilter{}, 0, 20)
	if err != nil {
		t.Fatalf("list inbox: %v", err)
	}
	if total != 1 || len(list) != 1 {
		t.Fatalf("got total=%d len=%d, want 1", total, len(list))
	}
	if got := list[0].FullCodePath; got != "/alice/hr/leave.form" {
		t.Fatalf("full_code_path = %q, want /alice/hr/leave.form", got)
	}
	if list[0].ThreadKey != "directory:/alice/hr" {
		t.Fatalf("thread_key = %q", list[0].ThreadKey)
	}
	if list[0].ScheduledTaskID != 12 || list[0].ScheduledExecutionID != 34 {
		t.Fatalf("scheduled ids = %d/%d, want 12/34", list[0].ScheduledTaskID, list[0].ScheduledExecutionID)
	}
	if list[0].WorkspaceSessionID != "session-1" {
		t.Fatalf("workspace_session_id = %q", list[0].WorkspaceSessionID)
	}
	if list[0].Files != "kageos/reports/leave.pdf" {
		t.Fatalf("files = %q", list[0].Files)
	}
	if list[0].SourceDisplay == nil {
		t.Fatal("source_display is nil")
	}
	if list[0].SourceDisplay.Name != "请假审批" || list[0].SourceDisplay.ParentName != "人事系统" || list[0].SourceDisplay.TemplateType != "form" {
		t.Fatalf("source_display = %#v", list[0].SourceDisplay)
	}
}

func TestListInboxThreadsGroupsByParentSource(t *testing.T) {
	repo := newTestMessageRepo(t)

	for _, title := range []string{"会议提醒 A", "会议提醒 B"} {
		_, err := repo.Create(context.Background(), dto.MessageSendMeta{
			From:              "system",
			SourceType:        "scheduled_task",
			SourcePath:        "/system/demos/meeting/notify.form",
			SourceTitle:       title,
			SourceParentPath:  "/system/demos/meeting",
			SourceParentTitle: "智能会议室",
			SourceRef:         "timer_task:9:execution:10",
		}, dto.MessageSendPayload{
			Title:   title,
			Content: "会议即将开始",
		}, []string{"bob"})
		if err != nil {
			t.Fatalf("create message: %v", err)
		}
	}

	threads, total, err := repo.ListInboxThreads(context.Background(), "bob", "", 0, 20)
	if err != nil {
		t.Fatalf("list inbox threads: %v", err)
	}
	if total != 1 || len(threads) != 1 {
		t.Fatalf("got total=%d len=%d, want 1", total, len(threads))
	}
	thread := threads[0]
	if thread.Title != "智能会议室" || thread.MessageCount != 2 || thread.UnreadCount != 2 {
		t.Fatalf("thread = %#v", thread)
	}
	if thread.Kind != "directory" || thread.Path != "/system/demos/meeting" {
		t.Fatalf("thread source = %#v", thread)
	}
	if thread.ScheduledTaskID != 9 || thread.ScheduledExecutionID != 10 {
		t.Fatalf("thread scheduled ids = %d/%d, want 9/10", thread.ScheduledTaskID, thread.ScheduledExecutionID)
	}

	messages, messageTotal, err := repo.ListInbox(context.Background(), "bob", InboxListFilter{ThreadKey: thread.Key}, 0, 20)
	if err != nil {
		t.Fatalf("list inbox by thread: %v", err)
	}
	if messageTotal != 2 || len(messages) != 2 {
		t.Fatalf("messages total=%d len=%d, want 2", messageTotal, len(messages))
	}
}

func TestListInboxBySourcePathAndSourceCounts(t *testing.T) {
	repo := newTestMessageRepo(t)

	fixtures := []struct {
		sourcePath string
		title      string
	}{
		{"/system/demos/meeting", "目录巡检提醒"},
		{"/system/demos/meeting/meeting_room_notify_soon.form", "会议即将开始提醒"},
		{"/system/demos/meeting/meeting_room_list.table", "会议室列表提醒"},
		{"/system/demos/other/notify.form", "其他系统提醒"},
	}
	for _, fixture := range fixtures {
		_, err := repo.Create(context.Background(), dto.MessageSendMeta{
			From:       "system",
			SourcePath: fixture.sourcePath,
		}, dto.MessageSendPayload{
			Title:   fixture.title,
			Content: fixture.title,
		}, []string{"bob"})
		if err != nil {
			t.Fatalf("create message %s: %v", fixture.sourcePath, err)
		}
	}

	counts, err := repo.ListSourceCounts(context.Background(), "bob", "")
	if err != nil {
		t.Fatalf("list source counts: %v", err)
	}
	countByPath := make(map[string]dto.MessageInboxSourceCount, len(counts))
	for _, count := range counts {
		countByPath[count.SourcePath] = count
	}
	if countByPath["/system/demos/meeting/meeting_room_notify_soon.form"].UnreadCount != 1 {
		t.Fatalf("function source count = %#v", countByPath["/system/demos/meeting/meeting_room_notify_soon.form"])
	}

	list, total, err := repo.ListInbox(context.Background(), "bob", InboxListFilter{
		SourcePath:      "/system/demos/meeting",
		IncludeChildren: true,
	}, 0, 20)
	if err != nil {
		t.Fatalf("list inbox by source path: %v", err)
	}
	if total != 3 || len(list) != 3 {
		t.Fatalf("source subtree total=%d len=%d, want 3", total, len(list))
	}

	direct, directTotal, err := repo.ListInbox(context.Background(), "bob", InboxListFilter{
		SourcePath: "/system/demos/meeting",
	}, 0, 20)
	if err != nil {
		t.Fatalf("list inbox by direct source path: %v", err)
	}
	if directTotal != 1 || len(direct) != 1 || direct[0].Title != "目录巡检提醒" {
		t.Fatalf("direct source list total=%d list=%#v", directTotal, direct)
	}
}

func TestListWorkspaceCountsGroupsByWorkspacePath(t *testing.T) {
	repo := newTestMessageRepo(t)

	readEntry, err := repo.Create(context.Background(), dto.MessageSendMeta{
		From:       "system",
		SourcePath: "/system/demos/inventory/supplier_list.table",
	}, dto.MessageSendPayload{
		Title:   "供应商提醒",
		Content: "供应商有更新",
	}, []string{"bob"})
	if err != nil {
		t.Fatalf("create read message: %v", err)
	}
	if err := repo.MarkRead(context.Background(), "bob", readEntry.ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}

	_, err = repo.Create(context.Background(), dto.MessageSendMeta{
		From:       "system",
		SourcePath: "/system/demos/inventory/order_list.table",
	}, dto.MessageSendPayload{
		Title:   "订单提醒",
		Content: "订单有更新",
	}, []string{"bob"})
	if err != nil {
		t.Fatalf("create unread message: %v", err)
	}

	_, err = repo.Create(context.Background(), dto.MessageSendMeta{
		From:       "system",
		SourcePath: "/alice/crm/customer_list.table",
	}, dto.MessageSendPayload{
		Title:   "客户提醒",
		Content: "客户有更新",
	}, []string{"bob"})
	if err != nil {
		t.Fatalf("create other workspace message: %v", err)
	}

	counts, err := repo.ListWorkspaceCounts(context.Background(), "bob", "")
	if err != nil {
		t.Fatalf("list workspace counts: %v", err)
	}
	countByWorkspace := make(map[string]dto.MessageInboxWorkspaceCount, len(counts))
	for _, count := range counts {
		countByWorkspace[count.WorkspaceKey] = count
	}

	systemDemos := countByWorkspace["/system/demos"]
	if systemDemos.WorkspaceUser != "system" || systemDemos.WorkspaceCode != "demos" {
		t.Fatalf("system workspace identity = %#v", systemDemos)
	}
	if systemDemos.MessageCount != 2 || systemDemos.UnreadCount != 1 {
		t.Fatalf("system workspace count = %#v, want total=2 unread=1", systemDemos)
	}
	if countByWorkspace["/alice/crm"].MessageCount != 1 || countByWorkspace["/alice/crm"].UnreadCount != 1 {
		t.Fatalf("alice workspace count = %#v", countByWorkspace["/alice/crm"])
	}

	unreadCounts, err := repo.ListWorkspaceCounts(context.Background(), "bob", "unread")
	if err != nil {
		t.Fatalf("list unread workspace counts: %v", err)
	}
	unreadByWorkspace := make(map[string]dto.MessageInboxWorkspaceCount, len(unreadCounts))
	for _, count := range unreadCounts {
		unreadByWorkspace[count.WorkspaceKey] = count
	}
	if unreadByWorkspace["/system/demos"].MessageCount != 1 || unreadByWorkspace["/system/demos"].UnreadCount != 1 {
		t.Fatalf("system unread workspace count = %#v", unreadByWorkspace["/system/demos"])
	}
}

func TestMarkReadAndUnreadCount(t *testing.T) {
	repo := newTestMessageRepo(t)
	entry, err := repo.Create(context.Background(), dto.MessageSendMeta{From: "alice"}, dto.MessageSendPayload{
		Title:   "hello",
		Content: "world",
	}, []string{"bob", "bob"})
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	count, err := repo.CountUnread(context.Background(), "bob")
	if err != nil {
		t.Fatalf("count unread: %v", err)
	}
	if count != 1 {
		t.Fatalf("unread count = %d, want 1", count)
	}
	if err := repo.MarkRead(context.Background(), "bob", entry.ID); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	count, err = repo.CountUnread(context.Background(), "bob")
	if err != nil {
		t.Fatalf("count unread after mark: %v", err)
	}
	if count != 0 {
		t.Fatalf("unread count after mark = %d, want 0", count)
	}
}

func TestMarkSourceReadMarksFunctionMessages(t *testing.T) {
	repo := newTestMessageRepo(t)
	functionPath := "/system/demos/meeting/meeting_room_notify_soon.form"
	siblingPath := "/system/demos/meeting/meeting_room_list.table"

	for i := 0; i < 25; i++ {
		_, err := repo.Create(context.Background(), dto.MessageSendMeta{
			From:       "system",
			SourcePath: functionPath,
		}, dto.MessageSendPayload{
			Title:   "函数提醒",
			Content: "会议即将开始",
		}, []string{"bob"})
		if err != nil {
			t.Fatalf("create function message %d: %v", i, err)
		}
	}
	_, err := repo.Create(context.Background(), dto.MessageSendMeta{
		From:       "system",
		SourcePath: siblingPath,
	}, dto.MessageSendPayload{
		Title:   "旁边函数提醒",
		Content: "不要被当前函数已读影响",
	}, []string{"bob"})
	if err != nil {
		t.Fatalf("create sibling message: %v", err)
	}

	if err := repo.MarkSourceRead(context.Background(), "bob", strings.TrimPrefix(functionPath, "/"), false); err != nil {
		t.Fatalf("mark source read: %v", err)
	}

	count, err := repo.CountUnread(context.Background(), "bob")
	if err != nil {
		t.Fatalf("count unread: %v", err)
	}
	if count != 1 {
		t.Fatalf("unread count after source mark = %d, want 1", count)
	}

	counts, err := repo.ListSourceCounts(context.Background(), "bob", "")
	if err != nil {
		t.Fatalf("list source counts: %v", err)
	}
	countByPath := make(map[string]dto.MessageInboxSourceCount, len(counts))
	for _, count := range counts {
		countByPath[count.SourcePath] = count
	}
	if got := countByPath[functionPath]; got.MessageCount != 25 || got.UnreadCount != 0 {
		t.Fatalf("function source count = %#v, want total=25 unread=0", got)
	}
	if got := countByPath[siblingPath]; got.MessageCount != 1 || got.UnreadCount != 1 {
		t.Fatalf("sibling source count = %#v, want total=1 unread=1", got)
	}
}

func TestMessageActionTokenViewAndReply(t *testing.T) {
	repo := newTestMessageRepo(t)
	entry, err := repo.Create(context.Background(), dto.MessageSendMeta{
		From:                  "alice",
		SourcePath:            "/alice/sales/orders.table",
		SourceTitle:           "订单列表",
		WorkspaceSessionID:    "session-1",
		WorkspaceSessionTitle: "订单跟进",
	}, dto.MessageSendPayload{
		Title:   "订单状态待确认",
		Content: "订单 A123 需要确认下一步。",
	}, []string{"bob"})
	if err != nil {
		t.Fatalf("create message: %v", err)
	}

	rawToken, tokenRow, err := repo.CreateActionToken(context.Background(), CreateActionTokenInput{
		MessageID:          entry.ID,
		RecipientUsername:  "bob",
		Channel:            "feishu",
		AllowedActions:     []string{"reply"},
		WorkspaceSessionID: entry.WorkspaceSessionID,
		ThreadKey:          entry.ThreadKey,
		SourcePath:         entry.SourcePath,
	})
	if err != nil {
		t.Fatalf("create action token: %v", err)
	}
	if rawToken == "" || tokenRow.TokenHash == "" || tokenRow.TokenHash == rawToken {
		t.Fatalf("token/hash not protected: raw=%q row=%#v", rawToken, tokenRow)
	}

	view, err := repo.GetActionView(context.Background(), rawToken, "/m")
	if err == nil {
		t.Fatalf("expected unauthenticated action view to fail")
	}
	view, err = repo.GetActionView(context.Background(), rawToken, "/m", "bob")
	if err != nil {
		t.Fatalf("get action view: %v", err)
	}
	if !view.CanReply || view.TokenStatus != "open" || view.Message.ID != entry.ID || view.WorkspaceSession != "session-1" {
		t.Fatalf("view = %#v", view)
	}

	reply, err := repo.SubmitActionReply(context.Background(), rawToken, "我在路上，先按原计划推进。", "reply", "bob")
	if err != nil {
		t.Fatalf("submit reply: %v", err)
	}
	if reply.ReplyMessageID != 0 || reply.Status != "submitted" {
		t.Fatalf("reply = %#v", reply)
	}
	if reply.SourcePath != "/alice/sales/orders.table" || reply.FullCodePath != "/alice/sales/orders.table" || reply.WorkspaceSessionID != "session-1" {
		t.Fatalf("reply context = %#v", reply)
	}
	if !strings.Contains(reply.WorkstationDraft, "订单 A123 需要确认下一步") ||
		!strings.Contains(reply.WorkstationDraft, "我在路上，先按原计划推进") {
		t.Fatalf("workstation draft = %q", reply.WorkstationDraft)
	}
	if !strings.Contains(reply.WorkstationDraft, "send_notification") ||
		!strings.Contains(reply.WorkstationDraft, "也看不到本轮工作台回复内容") ||
		!strings.Contains(reply.WorkstationDraft, "用户只能收到 send_notification 投递的消息通知") ||
		!strings.Contains(reply.WorkstationDraft, "必须使用 Markdown 格式") ||
		!strings.Contains(reply.WorkstationDraft, "content_type 使用 markdown") ||
		!strings.Contains(reply.WorkstationDraft, "通知正文禁止包含思考过程") ||
		!strings.Contains(reply.WorkstationDraft, "不能替代消息通知") {
		t.Fatalf("workstation draft missing mobile notification guardrails = %q", reply.WorkstationDraft)
	}
	if strings.Contains(reply.WorkstationDraft, "平台会自动") || strings.Contains(reply.WorkstationDraft, "自动回推") {
		t.Fatalf("workstation draft contains conflicting auto-push wording = %q", reply.WorkstationDraft)
	}

	thread, total, err := repo.ListInbox(context.Background(), "bob", InboxListFilter{ThreadKey: entry.ThreadKey}, 0, 20)
	if err != nil {
		t.Fatalf("list thread: %v", err)
	}
	if total != 1 || len(thread) != 1 || thread[0].ID != entry.ID {
		t.Fatalf("thread total=%d list=%#v", total, thread)
	}
	if _, err := repo.SubmitActionReply(context.Background(), rawToken, "重复回复", "reply", "bob"); err == nil {
		t.Fatal("expected duplicate submit to fail")
	}
}

func newTestMessageRepo(t *testing.T) *MessageRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := model.InitModels(db); err != nil {
		t.Fatalf("migrate message models: %v", err)
	}
	return NewMessageRepository(db)
}
