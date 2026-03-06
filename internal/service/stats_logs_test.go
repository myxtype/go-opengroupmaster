package service

import (
	"io"
	"log"
	"path/filepath"
	"testing"
	"time"

	"supervisor/internal/config"
	"supervisor/internal/model"
	"supervisor/internal/repository"

	"github.com/go-telegram/bot/models"
)

func TestGroupStatsByTGGroupID_WithSummary(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "stats.db")
	repo, err := repository.New("sqlite://"+dbPath, true)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}
	svc := New(repo, log.New(io.Discard, "", 0), &config.Config{})

	group, err := repo.UpsertGroup(&models.Chat{ID: -10001, Title: "统计测试群"})
	if err != nil {
		t.Fatalf("upsert group failed: %v", err)
	}
	u1, err := repo.UpsertUserFromTG(&models.User{ID: 101, Username: "alice"})
	if err != nil {
		t.Fatalf("upsert user1 failed: %v", err)
	}
	u2, err := repo.UpsertUserFromTG(&models.User{ID: 102, FirstName: "Bob", LastName: "Lee"})
	if err != nil {
		t.Fatalf("upsert user2 failed: %v", err)
	}
	if _, _, err := repo.AdjustPoints(group.ID, u1.ID, 20); err != nil {
		t.Fatalf("adjust points user1 failed: %v", err)
	}
	if _, _, err := repo.AdjustPoints(group.ID, u2.ID, 15); err != nil {
		t.Fatalf("adjust points user2 failed: %v", err)
	}

	now := time.Now()
	today := dateKeyAtTimezone(now, defaultGroupTimezoneOffsetMinutes)
	yesterday := dateKeyAtTimezone(now.Add(-24*time.Hour), defaultGroupTimezoneOffsetMinutes)
	oldDay := dateKeyAtTimezone(now.AddDate(0, 0, -40), defaultGroupTimezoneOffsetMinutes)

	events := []model.PointEvent{
		{GroupID: group.ID, UserID: u1.ID, DayKey: today, Type: pointsEventMessage, Delta: 10, CreatedAt: now},
		{GroupID: group.ID, UserID: u1.ID, DayKey: yesterday, Type: pointsEventMessage, Delta: 5, CreatedAt: now.Add(-24 * time.Hour)},
		{GroupID: group.ID, UserID: u2.ID, DayKey: today, Type: pointsEventMessage, Delta: 8, CreatedAt: now},
		{GroupID: group.ID, UserID: u2.ID, DayKey: oldDay, Type: pointsEventMessage, Delta: 7, CreatedAt: now.AddDate(0, 0, -40)},
		{GroupID: group.ID, UserID: u1.ID, DayKey: today, Type: pointsEventCheckin, Delta: 1, CreatedAt: now},
		{GroupID: group.ID, UserID: u2.ID, DayKey: oldDay, Type: pointsEventCheckin, Delta: 1, CreatedAt: now.AddDate(0, 0, -40)},
	}
	for _, event := range events {
		item := event
		if err := repo.CreatePointEvent(&item); err != nil {
			t.Fatalf("create point event failed: %v", err)
		}
	}

	if _, err := repo.CreateInviteEvent(&model.InviteEvent{
		GroupID:         group.ID,
		InviterTGUserID: 101,
		InviteeTGUserID: 201,
		Link:            "https://t.me/joinchat/test1",
		JoinedAt:        now,
	}); err != nil {
		t.Fatalf("create invite event1 failed: %v", err)
	}
	if _, err := repo.CreateInviteEvent(&model.InviteEvent{
		GroupID:         group.ID,
		InviterTGUserID: 102,
		InviteeTGUserID: 202,
		Link:            "https://t.me/joinchat/test2",
		JoinedAt:        now,
	}); err != nil {
		t.Fatalf("create invite event2 failed: %v", err)
	}
	if _, err := repo.CreateInviteEvent(&model.InviteEvent{
		GroupID:         group.ID,
		InviterTGUserID: 101,
		InviteeTGUserID: 203,
		Link:            "https://t.me/joinchat/test3",
		JoinedAt:        now.AddDate(0, 0, -40),
	}); err != nil {
		t.Fatalf("create invite event3 failed: %v", err)
	}

	stats, err := svc.GroupStatsByTGGroupID(group.TGGroupID, 10)
	if err != nil {
		t.Fatalf("group stats failed: %v", err)
	}

	if stats.DayKey != today {
		t.Fatalf("want day key %q, got %q", today, stats.DayKey)
	}
	if stats.TimezoneText != "UTC+8" {
		t.Fatalf("want timezone UTC+8, got %q", stats.TimezoneText)
	}
	if stats.PointsUsersTotal != 2 {
		t.Fatalf("want points users total=2, got %d", stats.PointsUsersTotal)
	}
	if stats.PointsTotal != 35 {
		t.Fatalf("want points total=35, got %d", stats.PointsTotal)
	}
	if stats.InviteTotal != 3 {
		t.Fatalf("want invite total=3, got %d", stats.InviteTotal)
	}
	if stats.MessageEventsTotal != 4 {
		t.Fatalf("want message events total=4, got %d", stats.MessageEventsTotal)
	}
	if stats.MessagePointsTotal != 30 {
		t.Fatalf("want message points total=30, got %d", stats.MessagePointsTotal)
	}
	if stats.MessageUsersTotal != 2 {
		t.Fatalf("want message users total=2, got %d", stats.MessageUsersTotal)
	}
	if stats.TodayMessagePoints != 18 {
		t.Fatalf("want today message points=18, got %d", stats.TodayMessagePoints)
	}
	if stats.TodayMessageUsers != 2 {
		t.Fatalf("want today message users=2, got %d", stats.TodayMessageUsers)
	}
	if stats.TodayCheckins != 1 {
		t.Fatalf("want today checkins=1, got %d", stats.TodayCheckins)
	}
	if stats.Recent7MessagePoints != 23 || stats.Recent7MessageUsers != 2 || stats.Recent7MessageEvents != 3 {
		t.Fatalf("unexpected recent7 message summary: points=%d users=%d events=%d", stats.Recent7MessagePoints, stats.Recent7MessageUsers, stats.Recent7MessageEvents)
	}
	if stats.Recent30MessagePoints != 23 || stats.Recent30MessageUsers != 2 || stats.Recent30MessageEvents != 3 {
		t.Fatalf("unexpected recent30 message summary: points=%d users=%d events=%d", stats.Recent30MessagePoints, stats.Recent30MessageUsers, stats.Recent30MessageEvents)
	}
	if stats.Recent7Checkins != 1 || stats.Recent30Checkins != 1 {
		t.Fatalf("unexpected checkins window: recent7=%d recent30=%d", stats.Recent7Checkins, stats.Recent30Checkins)
	}
	if stats.Recent7Invites != 2 || stats.Recent30Invites != 2 {
		t.Fatalf("unexpected invites window: recent7=%d recent30=%d", stats.Recent7Invites, stats.Recent30Invites)
	}

	if len(stats.TopUsers) != 2 {
		t.Fatalf("want 2 top users, got %d", len(stats.TopUsers))
	}
	if stats.TopUsers[0].DisplayName != "@alice" || stats.TopUsers[0].Points != 15 {
		t.Fatalf("unexpected top1: %+v", stats.TopUsers[0])
	}
	if stats.TopUsers[1].DisplayName != "Bob Lee" || stats.TopUsers[1].Points != 15 {
		t.Fatalf("unexpected top2: %+v", stats.TopUsers[1])
	}
}

func TestGroupStatsByTGGroupID_EmptyGroup(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "stats-empty.db")
	repo, err := repository.New("sqlite://"+dbPath, true)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}
	svc := New(repo, log.New(io.Discard, "", 0), &config.Config{})

	group, err := repo.UpsertGroup(&models.Chat{ID: -10002, Title: "空群"})
	if err != nil {
		t.Fatalf("upsert group failed: %v", err)
	}

	stats, err := svc.GroupStatsByTGGroupID(group.TGGroupID, 10)
	if err != nil {
		t.Fatalf("group stats failed: %v", err)
	}

	if stats.PointsUsersTotal != 0 || stats.PointsTotal != 0 {
		t.Fatalf("unexpected points summary: users=%d total=%d", stats.PointsUsersTotal, stats.PointsTotal)
	}
	if stats.InviteTotal != 0 {
		t.Fatalf("unexpected invite total: %d", stats.InviteTotal)
	}
	if stats.MessageEventsTotal != 0 || stats.MessagePointsTotal != 0 || stats.MessageUsersTotal != 0 {
		t.Fatalf("unexpected message summary: events=%d points=%d users=%d", stats.MessageEventsTotal, stats.MessagePointsTotal, stats.MessageUsersTotal)
	}
	if stats.TodayMessagePoints != 0 || stats.TodayMessageUsers != 0 || stats.TodayCheckins != 0 {
		t.Fatalf("unexpected today summary: points=%d users=%d checkins=%d", stats.TodayMessagePoints, stats.TodayMessageUsers, stats.TodayCheckins)
	}
	if stats.Recent7MessagePoints != 0 || stats.Recent7MessageUsers != 0 || stats.Recent7MessageEvents != 0 {
		t.Fatalf("unexpected recent7 summary: points=%d users=%d events=%d", stats.Recent7MessagePoints, stats.Recent7MessageUsers, stats.Recent7MessageEvents)
	}
	if stats.Recent30MessagePoints != 0 || stats.Recent30MessageUsers != 0 || stats.Recent30MessageEvents != 0 {
		t.Fatalf("unexpected recent30 summary: points=%d users=%d events=%d", stats.Recent30MessagePoints, stats.Recent30MessageUsers, stats.Recent30MessageEvents)
	}
	if stats.Recent7Checkins != 0 || stats.Recent30Checkins != 0 {
		t.Fatalf("unexpected checkins windows: recent7=%d recent30=%d", stats.Recent7Checkins, stats.Recent30Checkins)
	}
	if stats.Recent7Invites != 0 || stats.Recent30Invites != 0 {
		t.Fatalf("unexpected invites windows: recent7=%d recent30=%d", stats.Recent7Invites, stats.Recent30Invites)
	}
	if len(stats.TopUsers) != 0 {
		t.Fatalf("want empty top users, got %d", len(stats.TopUsers))
	}
	if stats.DayKey == "" {
		t.Fatalf("day key should not be empty")
	}
	if stats.TimezoneText != "UTC+8" {
		t.Fatalf("want timezone UTC+8, got %q", stats.TimezoneText)
	}
}

func TestListLogsByTGGroupID_WithDisplayNames(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "logs.db")
	repo, err := repository.New("sqlite://"+dbPath, true)
	if err != nil {
		t.Fatalf("new repo failed: %v", err)
	}
	svc := New(repo, log.New(io.Discard, "", 0), &config.Config{})

	group, err := repo.UpsertGroup(&models.Chat{ID: -10003, Title: "日志测试群"})
	if err != nil {
		t.Fatalf("upsert group failed: %v", err)
	}
	operator, err := repo.UpsertUserFromTG(&models.User{ID: 201, Username: "alice"})
	if err != nil {
		t.Fatalf("upsert operator failed: %v", err)
	}
	target, err := repo.UpsertUserFromTG(&models.User{ID: 202, FirstName: "Bob", LastName: "Lee"})
	if err != nil {
		t.Fatalf("upsert target failed: %v", err)
	}
	if err := repo.CreateLog(group.ID, "mute", operator.ID, target.ID); err != nil {
		t.Fatalf("create log failed: %v", err)
	}
	if err := repo.CreateLog(group.ID, "banned_word_delete", 0, target.ID); err != nil {
		t.Fatalf("create system log failed: %v", err)
	}

	page, err := svc.ListLogsByTGGroupID(group.TGGroupID, 1, 10, "all")
	if err != nil {
		t.Fatalf("list logs failed: %v", err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("want 2 log items, got %d", len(page.Items))
	}
	if page.Items[0].OperatorDisplayName != "" {
		t.Fatalf("want empty operator for system log, got %q", page.Items[0].OperatorDisplayName)
	}
	if page.Items[0].TargetDisplayName != "Bob Lee" {
		t.Fatalf("want target display name Bob Lee, got %q", page.Items[0].TargetDisplayName)
	}
	if page.Items[1].OperatorDisplayName != "@alice" {
		t.Fatalf("want operator display name @alice, got %q", page.Items[1].OperatorDisplayName)
	}
	if page.Items[1].TargetDisplayName != "Bob Lee" {
		t.Fatalf("want target display name Bob Lee, got %q", page.Items[1].TargetDisplayName)
	}
}
