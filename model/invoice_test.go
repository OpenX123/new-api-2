package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// migrateInvoiceTables 幂等迁移本功能所需表（TestMain 的列表不含发票表）
func migrateInvoiceTables(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(&Invoice{}, &InvoiceOrder{}, &InvoiceFile{}, &TopUp{}, &SubscriptionOrder{}))
}

func clearInvoiceTables(t *testing.T) {
	t.Helper()
	migrateInvoiceTables(t)
	require.NoError(t, DB.Exec("DELETE FROM invoice_files").Error)
	require.NoError(t, DB.Exec("DELETE FROM invoice_orders").Error)
	require.NoError(t, DB.Exec("DELETE FROM invoices").Error)
	require.NoError(t, DB.Exec("DELETE FROM top_ups").Error)
	require.NoError(t, DB.Exec("DELETE FROM subscription_orders").Error)
}

func TestInvoiceOrderUniqueIndex(t *testing.T) {
	clearInvoiceTables(t)
	inv := &Invoice{UserId: 1, InvoiceNo: "INV-T1", TitleType: InvoiceTitleTypePersonal,
		TitleName: "张三", Email: "a@b.c", Money: 10, Status: InvoiceStatusPending, CreateTime: 1}
	require.NoError(t, DB.Create(inv).Error)
	o1 := &InvoiceOrder{InvoiceId: inv.Id, OrderType: InvoiceOrderTypeTopup, OrderId: 100, TradeNo: "T100", Money: 10}
	require.NoError(t, DB.Create(o1).Error)
	// 同一订单再次挂接必须违反唯一索引
	o2 := &InvoiceOrder{InvoiceId: inv.Id + 999, OrderType: InvoiceOrderTypeTopup, OrderId: 100, TradeNo: "T100", Money: 10}
	require.Error(t, DB.Create(o2).Error)
}
