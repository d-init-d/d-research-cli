package search

import (
	"context"
	"fmt"
	"testing"

	"github.com/d-init-d/d-research-cli/internal/config"
)

func TestFallbackOrdering(t *testing.T) {
	p1 := &FakeProvider{IDValue: "a", Err: fmt.Errorf("fail")}
	p2 := &FakeProvider{IDValue: "b", Results: []Result{{Title: "ok", URL: "https://example.com"}}}
	mgr := &Manager{providers: []Provider{p1, p2}, strategy: "fallback"}
	res, id, err := mgr.Search(context.Background(), "q")
	if err != nil || id != "b" || len(res) != 1 {
		t.Fatalf("res=%v id=%s err=%v", res, id, err)
	}
}

func TestFakeProviderStatuses(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		p    *FakeProvider
		hint string
	}{
		{&FakeProvider{IDValue: "x", StatusCode: 401}, "auth_failed"},
		{&FakeProvider{IDValue: "x", StatusCode: 429}, "rate_limit"},
	} {
		rep := TestProvider(ctx, tc.p)
		if rep.Available {
			t.Fatal("expected unavailable")
		}
	}
	_ = config.SearchProvider{}
}