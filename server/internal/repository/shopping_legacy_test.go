package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	shoppingapp "github.com/bensuskins/family-hub/internal/application/shopping"
	shoppingdomain "github.com/bensuskins/family-hub/internal/domain/shopping"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/google/uuid"
)

func TestShoppingRestoresLegacyStateWithoutRepair(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `INSERT INTO shopping_items(id,name,purchased,restocked,version) VALUES('legacy','',0,1,4)`); err != nil {
		t.Fatal(err)
	}
	service := shoppingapp.New(repository.NewShoppingRepository(database, repository.NewShoppingInventory), uuid.NewString, time.Now)
	items, err := service.List(ctx)
	if err != nil || len(items) != 1 || items[0].Name != "" || !items[0].Restocked || items[0].Version != 4 {
		t.Fatalf("stored item: %+v %v", items, err)
	}
	if _, err := service.Restock(ctx, "legacy", shoppingdomain.Restock{PurchaseID: "old"}); !errors.Is(err, shoppingdomain.ErrConflict) {
		t.Fatalf("restock error %v", err)
	}
	updated, err := service.Update(ctx, "legacy", shoppingdomain.Change{Version: 4})
	if err != nil || updated.Name != "" || !updated.Restocked || updated.Version != 5 {
		t.Fatalf("legacy update: %+v %v", updated, err)
	}
	var name string
	var restocked bool
	var version int
	if err := database.QueryRowContext(ctx, "SELECT name,restocked,version FROM shopping_items WHERE id='legacy'").Scan(&name, &restocked, &version); err != nil {
		t.Fatal(err)
	}
	if name != "" || !restocked || version != 5 {
		t.Fatalf("record rewritten: %q %v %d", name, restocked, version)
	}
}
