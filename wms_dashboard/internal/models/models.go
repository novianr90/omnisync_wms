package models

import (
	"time"

	"gorm.io/gorm"
)

// Product represents the Master Product Catalog (Static Definition)
type Product struct {
	ID          string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	SKU         string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"sku"`
	Name        string         `gorm:"type:varchar(255);not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Category    string         `gorm:"type:varchar(100)" json:"category"`
	Price       float64        `gorm:"type:decimal(12,2);default:0.00" json:"price"`
	IsBundle    bool           `gorm:"type:boolean;default:false" json:"is_bundle"`
	UoMID       string         `gorm:"type:varchar(36);index;column:uom_id" json:"uom_id"`
	UoM         UoM            `gorm:"foreignKey:UoMID" json:"uom,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Warehouse represents a Master Warehouse
type Warehouse struct {
	ID        string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	Code      string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"` // e.g. "WH-MAIN"
	Name      string         `gorm:"type:varchar(100);not null" json:"name"`
	Address   string         `gorm:"type:text" json:"address"`
	IsActive  bool           `gorm:"type:boolean;default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Locator represents a physical location inside a Warehouse (Aisle, Shelf, Level)
type Locator struct {
	ID          string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	WarehouseID string         `gorm:"type:varchar(36);not null;index" json:"warehouse_id"`
	Zone        string         `gorm:"type:varchar(20);not null" json:"zone"`  // e.g. "Zone-A"
	Aisle       string         `gorm:"type:varchar(20);not null" json:"aisle"` // e.g. "Aisle-14"
	Shelf       string         `gorm:"type:varchar(20);not null" json:"shelf"` // e.g. "Shelf-3"
	Level       string         `gorm:"type:varchar(20);not null" json:"level"` // e.g. "Level-2"
	Code        string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"code"` // combined unique code: "WH-MAIN-A-14-3-2"
	IsActive    bool           `gorm:"type:boolean;default:true" json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Preloads
	Warehouse Warehouse `gorm:"foreignKey:WarehouseID" json:"warehouse,omitempty"`
}

// Storage represents the Inventory Balance Master (Source of Truth for Stock)
type Storage struct {
	ID           string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	ProductID    string    `gorm:"type:varchar(36);not null;index" json:"product_id"`
	LocatorID    string    `gorm:"type:varchar(36);not null;index" json:"locator_id"`
	BatchNumber  string    `gorm:"type:varchar(100);not null;index" json:"batch_number"` // FIFO tracking
	SerialNumber string    `gorm:"type:varchar(100);index" json:"serial_number"`         // Optional individual serial
	ReceivedAt   time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"received_at"`          // Used for FIFO ordering
	QtyOnHand    int       `gorm:"type:int;default:0" json:"qty_on_hand"`                // Physical stock sitting on shelf
	QtyReserved  int       `gorm:"type:int;default:0" json:"qty_reserved"`               // Reserved stock for open movements
	QtyOnHold    int       `gorm:"type:int;default:0" json:"qty_on_hold"`                // Stock frozen by QC Hold
	UpdatedAt    time.Time `json:"updated_at"`

	// Preloads
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Locator Locator `gorm:"foreignKey:LocatorID" json:"locator,omitempty"`
}

// AvailableQty returns the quantity in this storage lot that is not reserved or on hold
func (s Storage) AvailableQty() int {
	return s.QtyOnHand - s.QtyReserved - s.QtyOnHold
}

// InventoryMovement represents an inbound/outbound/internal ticket header
type InventoryMovement struct {
	ID                 string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	DocumentNo         string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"document_no"` // e.g. "MOV-2026-0001"
	MovementType       string    `gorm:"type:varchar(20);not null" json:"movement_type"`           // INBOUND, OUTBOUND, INTERNAL, RTV
	IsCrossDock        bool      `gorm:"type:boolean;default:false" json:"is_cross_dock"`          // Flag for cross docking
	Status             string    `gorm:"type:varchar(20);default:'OPEN'" json:"status"`            // OPEN, IN_PROGRESS, RECEIPT, JOURNALED, COMPLETED, REJECTED
	CreatedBy          string    `gorm:"type:varchar(36);not null" json:"created_by"`             // User ID from JWT
	AssignedOperatorID string    `gorm:"type:varchar(36)" json:"assigned_operator_id"`            // Operator assigned (User ID)
	Remarks            string    `gorm:"type:text" json:"remarks"`
	RejectionReason    string    `gorm:"type:text" json:"rejection_reason,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`

	// Nested Preloads
	Lines []InventoryMovementLine `gorm:"foreignKey:MovementID" json:"lines,omitempty"`
}

// InventoryMovementLine represents individual items inside a movement ticket
type InventoryMovementLine struct {
	ID                string `gorm:"type:varchar(36);primaryKey" json:"id"`
	MovementID        string `gorm:"type:varchar(36);not null;index" json:"movement_id"`
	ProductID         string `gorm:"type:varchar(36);not null;index" json:"product_id"`
	BatchNumber       string `gorm:"type:varchar(100)" json:"batch_number,omitempty"`
	FromLocatorID     string `gorm:"type:varchar(36);index" json:"from_locator_id,omitempty"` // For Outbound/Internal
	ToLocatorID       string `gorm:"type:varchar(36);index" json:"to_locator_id,omitempty"`   // For Inbound/Internal
	RequestedQuantity int    `gorm:"type:int;not null" json:"requested_quantity"`
	ActualQuantity    int    `gorm:"type:int;default:0" json:"actual_quantity"`
	IsFromHold        bool   `gorm:"type:boolean;default:false" json:"is_from_hold"`

	// Preloads
	Product     Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	FromLocator *Locator `gorm:"foreignKey:FromLocatorID" json:"from_locator,omitempty"`
	ToLocator   *Locator `gorm:"foreignKey:ToLocatorID" json:"to_locator,omitempty"`
}

// UoM represents a Master Unit of Measure
type UoM struct {
	ID          string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	Code        string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"` // e.g. "kg", "pack"
	Name        string         `gorm:"type:varchar(100);not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	CreatedAt   time.Time      `json:"created_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides GORM's default naming (which would produce "uo_ms").
func (UoM) TableName() string { return "uoms" }

// UoMConversion represents a Dynamic Unit of Measure Conversion Formula
type UoMConversion struct {
	ID             string         `gorm:"type:varchar(36);primaryKey" json:"id"`
	ProductID      string         `gorm:"type:varchar(36);index" json:"product_id"`           // Optional: product-specific conversion, global if empty
	FromUoMID      string         `gorm:"type:varchar(36);not null;index;column:from_uom_id" json:"from_uom_id"`
	ToUoMID        string         `gorm:"type:varchar(36);not null;index;column:to_uom_id" json:"to_uom_id"`
	MultiplyFactor float64        `gorm:"type:decimal(12,6);default:1.0" json:"multiply_factor"` // conversion formula factor
	CreatedAt      time.Time      `json:"created_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Preloads
	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	FromUo  UoM      `gorm:"foreignKey:FromUoMID" json:"from_uom,omitempty"`
	ToUo    UoM      `gorm:"foreignKey:ToUoMID" json:"to_uom,omitempty"`
}

// TableName overrides GORM's default naming (which would produce "uo_m_conversions").
func (UoMConversion) TableName() string { return "uom_conversions" }

// InventoryAdjustment represents a stock correction ticket
type InventoryAdjustment struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	DocumentNo  string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"document_no"`
	Status      string    `gorm:"type:varchar(20);default:'OPEN'" json:"status"` // OPEN, JOURNALED, REJECTED
	ReasonCode  string    `gorm:"type:varchar(50);not null" json:"reason_code"`  // DAMAGED, LOST, FOUND, EXPIRED
	Remarks     string    `gorm:"type:text" json:"remarks"`
	CreatedBy   string    `gorm:"type:varchar(36);not null" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Nested Preloads
	Lines []InventoryAdjustmentLine `gorm:"foreignKey:AdjustmentID" json:"lines,omitempty"`
}

// InventoryAdjustmentLine represents individual stock changes
type InventoryAdjustmentLine struct {
	ID           string `gorm:"type:varchar(36);primaryKey" json:"id"`
	AdjustmentID string `gorm:"type:varchar(36);not null;index" json:"adjustment_id"`
	ProductID    string `gorm:"type:varchar(36);not null;index" json:"product_id"`
	LocatorID    string `gorm:"type:varchar(36);not null;index" json:"locator_id"`
	QtyDelta     int    `gorm:"type:int;not null" json:"qty_delta"` // Negative to deduct, Positive to add
	
	// Preloads
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Locator Locator `gorm:"foreignKey:LocatorID" json:"locator,omitempty"`
}

// InventoryKitting represents a light assembly / bundling ticket
type InventoryKitting struct {
	ID                string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	DocumentNo        string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"document_no"`
	Status            string    `gorm:"type:varchar(20);default:'OPEN'" json:"status"` // OPEN, JOURNALED, REJECTED
	FinishedProductID string    `gorm:"type:varchar(36);not null;index" json:"finished_product_id"`
	FinishedLocatorID string    `gorm:"type:varchar(36);not null" json:"finished_locator_id"`
	FinishedQty       int       `gorm:"type:int;not null" json:"finished_qty"`
	Remarks           string    `gorm:"type:text" json:"remarks"`
	CreatedBy         string    `gorm:"type:varchar(36);not null" json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Preloads
	FinishedProduct Product                `gorm:"foreignKey:FinishedProductID" json:"finished_product,omitempty"`
	FinishedLocator Locator                `gorm:"foreignKey:FinishedLocatorID" json:"finished_locator,omitempty"`
	ComponentLines  []InventoryKittingLine `gorm:"foreignKey:KittingID" json:"component_lines,omitempty"`
}

// InventoryKittingLine represents the components consumed to build the bundle
type InventoryKittingLine struct {
	ID           string `gorm:"type:varchar(36);primaryKey" json:"id"`
	KittingID    string `gorm:"type:varchar(36);not null;index" json:"kitting_id"`
	ProductID    string `gorm:"type:varchar(36);not null;index" json:"product_id"`
	LocatorID    string `gorm:"type:varchar(36);not null;index" json:"locator_id"`
	ConsumedQty  int    `gorm:"type:int;not null" json:"consumed_qty"` // Must be > 0

	// Preloads
	Product Product `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Locator Locator `gorm:"foreignKey:LocatorID" json:"locator,omitempty"`
}

// QCHold represents a Quality Control stock freeze record
type QCHold struct {
	ID         string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	DocumentNo string     `gorm:"type:varchar(50);uniqueIndex" json:"document_no"`
	StorageID  string     `gorm:"type:varchar(36);not null;index" json:"storage_id"`
	Qty        int        `gorm:"type:int;not null" json:"qty"`
	Reason     string     `gorm:"type:varchar(50);not null" json:"reason"`     // DAMAGED, INVESTIGATION, EXPIRED, OTHER
	Status     string     `gorm:"type:varchar(20);default:'ACTIVE'" json:"status"` // ACTIVE, RELEASED
	Notes      string     `gorm:"type:text" json:"notes"`
	CreatedBy  string     `gorm:"type:varchar(36);not null" json:"created_by"`
	ReleasedBy string     `gorm:"type:varchar(36)" json:"released_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	ReleasedAt *time.Time `json:"released_at,omitempty"`

	// Preloads
	Storage Storage `gorm:"foreignKey:StorageID" json:"storage,omitempty"`
}

// SequenceGenerator represents a dynamic sequence config and offset for a target context
type SequenceGenerator struct {
	ID            string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UsageTable    string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"usage_table"` // e.g. 'inventory_movements'
	Prefix        string    `gorm:"type:varchar(10);not null" json:"prefix"`                  // e.g. 'MOV'
	FiscalYear    int       `gorm:"type:int;not null" json:"fiscal_year"`                     // e.g. 2026
	CurrentNumber int       `gorm:"type:int;not null;default:1" json:"current_number"`
	NumberLength  int       `gorm:"type:int;not null;default:5" json:"number_length"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (SequenceGenerator) TableName() string { return "sequence_generators" }

// Account represents an Accounting Chart of Accounts
type Account struct {
	AccountNo   string    `gorm:"type:varchar(50);primaryKey" json:"account_no"`
	AccountName string    `gorm:"type:varchar(100);not null" json:"account_name"`
	AccountType string    `gorm:"type:varchar(50);not null" json:"account_type"` // ASSET, LIABILITY, EQUITY, REVENUE, EXPENSE
	CreatedAt   time.Time `json:"created_at"`
}

const (
	AccInventoryAsset      = "11000" // Raw Materials & General Inventory
	AccFinishedGoods       = "11010"
	AccWIP                 = "11020" // Work In Progress (Kitting)
	AccAccountsPayable     = "21000" // A/P (GRNI)
	AccCOGS                = "51000" // Cost of Goods Sold
	AccInventoryAdjustment = "51010" // Inventory Adjustment Expense
)

// InventoryLedger tracks atomic stock mutations for audit and valuation
type InventoryLedger struct {
	ID              string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	TransactionDate time.Time `gorm:"not null;index" json:"transaction_date"`
	ProductID       string    `gorm:"type:varchar(36);not null;index" json:"product_id"`
	LocatorID       string    `gorm:"type:varchar(36);not null;index" json:"locator_id"`
	BatchNumber     string    `gorm:"type:varchar(100);not null;index" json:"batch_number"`
	TransactionType string    `gorm:"type:varchar(50);not null;index" json:"transaction_type"` // INBOUND, OUTBOUND, TRANSFER, RTV, KITTING, ADJUSTMENT, HOLD, RELEASE
	DocumentNo      string    `gorm:"type:varchar(50);not null;index" json:"document_no"`
	QtyChange       int       `gorm:"type:int;not null" json:"qty_change"`
	BatchBalance    int       `gorm:"type:int;not null" json:"batch_balance"`
	AccountNo       *string   `gorm:"type:varchar(50);index" json:"account_no,omitempty"`               // Reference to Account for Inventory Valuation
	ContraAccountNo *string   `gorm:"type:varchar(50);index" json:"contra_account_no,omitempty"`        // Reference to balancing Account (COGS, Adjustment, etc)
	CreatedBy       string    `gorm:"type:varchar(36);not null" json:"created_by"`

	// Preloads
	Product       Product  `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	Locator       Locator  `gorm:"foreignKey:LocatorID" json:"locator,omitempty"`
	Account       *Account `gorm:"foreignKey:AccountNo;references:AccountNo" json:"account,omitempty"`
	ContraAccount *Account `gorm:"foreignKey:ContraAccountNo;references:AccountNo" json:"contra_account,omitempty"`
}

// TableName overrides GORM's default naming
func (InventoryLedger) TableName() string { return "inventory_ledgers" }
