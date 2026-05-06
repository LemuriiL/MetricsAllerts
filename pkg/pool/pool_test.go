package pool

import (
	"testing"
)

type testItem struct {
	Count      int
	Name       string
	Items      []string
	Meta       map[string]string
	ResetCalls int
}

func (v *testItem) Reset() {
	if v == nil {
		return
	}

	v.Count = 0
	v.Name = ""
	v.Items = v.Items[:0]
	clear(v.Meta)
	v.ResetCalls++
}

func TestPoolGetUsesFactory(t *testing.T) {
	p := New[*testItem](func() *testItem {
		return &testItem{
			Items: make([]string, 0),
			Meta:  make(map[string]string),
		}
	})

	item := p.Get()

	if item == nil {
		t.Fatal("expected item")
	}

	if item.Items == nil {
		t.Fatal("expected items slice")
	}

	if item.Meta == nil {
		t.Fatal("expected meta map")
	}
}

func TestPoolPutResetsObject(t *testing.T) {
	p := New[*testItem]()

	item := &testItem{
		Count: 10,
		Name:  "metric",
		Items: []string{"a", "b"},
		Meta:  map[string]string{"key": "value"},
	}

	p.Put(item)

	if item.Count != 0 {
		t.Fatalf("expected count reset, got %d", item.Count)
	}

	if item.Name != "" {
		t.Fatalf("expected name reset, got %s", item.Name)
	}

	if len(item.Items) != 0 {
		t.Fatalf("expected empty items, got %d", len(item.Items))
	}

	if len(item.Meta) != 0 {
		t.Fatalf("expected empty meta, got %d", len(item.Meta))
	}

	if item.ResetCalls != 1 {
		t.Fatalf("expected reset calls 1, got %d", item.ResetCalls)
	}
}

func TestPoolGetReturnsPreviouslyPutObject(t *testing.T) {
	p := New[*testItem]()

	item := &testItem{
		Count: 1,
		Name:  "metric",
		Items: []string{"a"},
		Meta:  map[string]string{"key": "value"},
	}

	p.Put(item)

	got := p.Get()

	if got != item {
		t.Fatal("expected previously put item")
	}

	if got.Count != 0 {
		t.Fatalf("expected count reset, got %d", got.Count)
	}

	if got.Name != "" {
		t.Fatalf("expected name reset, got %s", got.Name)
	}

	if len(got.Items) != 0 {
		t.Fatalf("expected empty items, got %d", len(got.Items))
	}

	if len(got.Meta) != 0 {
		t.Fatalf("expected empty meta, got %d", len(got.Meta))
	}
}

func TestPoolPutNilDoesNothing(t *testing.T) {
	p := New[*testItem]()

	p.Put(nil)

	got := p.Get()

	if got != nil {
		t.Fatal("expected nil")
	}
}
