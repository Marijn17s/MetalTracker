package domain

import "time"

type AssetClass string

const (
	AssetClassPreciousMetal AssetClass = "precious_metal"
	AssetClassCrypto        AssetClass = "crypto"
	AssetClassIndexFund     AssetClass = "index_fund"
)

type MetalSymbol string

const (
	MetalGold   MetalSymbol = "XAU"
	MetalSilver MetalSymbol = "XAG"
)

type Form string

const (
	FormCoin  Form = "coin"
	FormBar   Form = "bar"
	FormOther Form = "other"
)

type WeightUnit string

const (
	WeightUnitGram      WeightUnit = "g"
	WeightUnitKilogram  WeightUnit = "kg"
	WeightUnitTroyOz    WeightUnit = "troy_oz"
)

type Currency string

const (
	CurrencyEUR Currency = "EUR"
	CurrencyUSD Currency = "USD"
	CurrencyCHF Currency = "CHF"
)

type UnitStatus string

const (
	UnitStatusHeld UnitStatus = "held"
	UnitStatusSold UnitStatus = "sold"
)

type PriceSource string

const (
	PriceSourceMetalpriceAPI PriceSource = "metalpriceapi"
	PriceSourceMiddleman     PriceSource = "middleman"
)

// DefaultMiddlemanBaseURL is prefilled for new vaults; users can change it to their own host.
const DefaultMiddlemanBaseURL = "https://metaltracker.moose-vimba.ts.net"

type SpotPriceUnit string

const (
	SpotPriceUnitGram     SpotPriceUnit = "g"
	SpotPriceUnitTroyOz   SpotPriceUnit = "troy_oz"
	SpotPriceUnitKilogram SpotPriceUnit = "kilogram"
)

type UITheme string

const (
	UIThemeDark  UITheme = "dark"
	UIThemeLight UITheme = "light"
)

type ProductKey struct {
	AssetClass  AssetClass  `json:"assetClass"`
	Metal       MetalSymbol `json:"metal"`
	Form        Form        `json:"form"`
	WeightGrams float64     `json:"weightGrams"`
	Purity      float64     `json:"purity"`
	Brand       string      `json:"brand"`
	ProductName string      `json:"productName"`
}

func (key ProductKey) String() string {
	return BuildProductKeyString(key)
}

func BuildProductKeyString(key ProductKey) string {
	return string(key.AssetClass) + "|" +
		string(key.Metal) + "|" +
		string(key.Form) + "|" +
		formatFloat(key.WeightGrams) + "|" +
		formatFloat(key.Purity) + "|" +
		key.Brand + "|" +
		key.ProductName
}

func formatFloat(value float64) string {
	return trimFloat(value)
}

type Investment struct {
	ID          string   `json:"id"`
	PurchasedAt string   `json:"purchasedAt"`
	Currency    Currency `json:"currency"`
	Notes       string   `json:"notes"`
	Dealer      string   `json:"dealer"`
	CreatedAt   string   `json:"createdAt"`
}

type InvestmentLineInput struct {
	AssetClass         AssetClass  `json:"assetClass"`
	Metal              MetalSymbol `json:"metal"`
	Form               Form        `json:"form"`
	Weight             float64     `json:"weight"`
	WeightUnit         WeightUnit  `json:"weightUnit"`
	Purity             float64     `json:"purity"`
	Brand              string      `json:"brand"`
	ProductName        string      `json:"productName"`
	Quantity           int         `json:"quantity"`
	TotalPurchasePrice float64     `json:"totalPurchasePrice"`
	TotalSpotWorth     float64     `json:"totalSpotWorth"`
	SpotWorthProvided  bool        `json:"spotWorthProvided"`
	StorageLocation string `json:"storageLocation"`
	Condition       string `json:"condition"`
	MintageYear     int    `json:"mintageYear"`
}

type CreateInvestmentRequest struct {
	PurchasedAt string                `json:"purchasedAt"`
	Currency    Currency              `json:"currency"`
	Notes       string                `json:"notes"`
	Dealer      string                `json:"dealer"`
	Lines       []InvestmentLineInput `json:"lines"`
}

type HoldingUnit struct {
	ID                  string      `json:"id"`
	InvestmentID        string      `json:"investmentId"`
	AssetClass          AssetClass  `json:"assetClass"`
	Metal               MetalSymbol `json:"metal"`
	Form                Form        `json:"form"`
	WeightGrams         float64     `json:"weightGrams"`
	Purity              float64     `json:"purity"`
	Brand               string      `json:"brand"`
	ProductName         string      `json:"productName"`
	ProductKey          string      `json:"productKey"`
	Currency            Currency    `json:"currency"`
	PurchasePrice       float64     `json:"purchasePrice"`
	SpotWorthAtPurchase float64     `json:"spotWorthAtPurchase"`
	PurchasedAt         string      `json:"purchasedAt"`
	Status              UnitStatus  `json:"status"`
	SoldAt              string      `json:"soldAt,omitempty"`
	SalePrice           *float64    `json:"salePrice,omitempty"`
	Notes               string      `json:"notes"`
	Dealer              string      `json:"dealer"`
	StorageLocation     string      `json:"storageLocation"`
	Condition           string      `json:"condition"`
	MintageYear         int         `json:"mintageYear"`
	DeletedAt           string      `json:"deletedAt,omitempty"`
}

const (
	AttachmentOwnerUnit       = "unit"
	AttachmentOwnerInvestment = "investment"
	AttachmentKindPhoto       = "photo"
	AttachmentKindReceipt     = "receipt"
)

type Attachment struct {
	ID           string `json:"id"`
	OwnerType    string `json:"ownerType"`
	OwnerID      string `json:"ownerId"`
	Kind         string `json:"kind"`
	Filename     string `json:"filename"`
	ContentType  string `json:"contentType"`
	RelativePath string `json:"relativePath"`
	CreatedAt    string `json:"createdAt"`
}

type AttachmentBytes struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"contentType"`
	DataBase64  string `json:"dataBase64"`
}

type DealerSummary struct {
	Name          string  `json:"name"`
	PurchaseCount int     `json:"purchaseCount"`
	UnitCount     int     `json:"unitCount"`
	TotalSpent    float64 `json:"totalSpent"`
	Currency      Currency `json:"currency"`
}

func ParseDate(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	return time.Parse("2006-01-02", value)
}

type UnitValuation struct {
	HoldingUnit
	CurrentSpotWorth       float64  `json:"currentSpotWorth"`
	PremiumPaid            float64  `json:"premiumPaid"`
	MetalDelta             float64  `json:"metalDelta"`
	TotalProfit            float64  `json:"totalProfit"`
	TotalProfitPct         float64  `json:"totalProfitPct"`
	IsRealized             bool     `json:"isRealized"`
	FineWeightGrams        float64  `json:"fineWeightGrams"`
	ValuationApproximate   bool     `json:"valuationApproximate"`
	DisplayCurrency        Currency `json:"displayCurrency,omitempty"`
	FxRateToDisplay        float64  `json:"fxRateToDisplay,omitempty"`
	BreakEvenSpotPerKg float64  `json:"breakEvenSpotPerKg,omitempty"`
	DaysHeld               int      `json:"daysHeld,omitempty"`
	AnnualizedReturnPct    float64  `json:"annualizedReturnPct,omitempty"`
}

type GroupedHolding struct {
	ProductKey              string      `json:"productKey"`
	AssetClass              AssetClass  `json:"assetClass"`
	Metal                   MetalSymbol `json:"metal"`
	Form                    Form        `json:"form"`
	WeightGrams             float64     `json:"weightGrams"`
	Purity                  float64     `json:"purity"`
	Brand                   string      `json:"brand"`
	ProductName             string      `json:"productName"`
	Currency                Currency    `json:"currency"`
	DisplayCurrency         Currency    `json:"displayCurrency"`
	HeldCount               int         `json:"heldCount"`
	SoldCount               int         `json:"soldCount"`
	TotalCount              int         `json:"totalCount"`
	TotalWeightGrams        float64     `json:"totalWeightGrams"`
	TotalFineWeightGrams    float64     `json:"totalFineWeightGrams"`
	TotalPurchasePrice      float64     `json:"totalPurchasePrice"`
	TotalCurrentWorth       float64     `json:"totalCurrentWorth"`
	TotalProfit             float64     `json:"totalProfit"`
	TotalProfitPct          float64     `json:"totalProfitPct"`
	TotalRealizedProfit        float64 `json:"totalRealizedProfit"`
	TotalUnrealizedProfit      float64 `json:"totalUnrealizedProfit"`
	ValuationApproximate       bool    `json:"valuationApproximate"`
	BreakEvenSpotPerKg         float64 `json:"breakEvenSpotPerKg,omitempty"`
	AvgCostPerKgFine           float64 `json:"avgCostPerKgFine,omitempty"`
	HeldPurchasePrice          float64 `json:"heldPurchasePrice,omitempty"`
	HeldFineWeightGrams        float64 `json:"heldFineWeightGrams,omitempty"`
}

type HoldingsFilter struct {
	Search    string   `json:"search"`
	Metals    []string `json:"metals"`
	Forms     []string `json:"forms"`
	Brands    []string `json:"brands"`
	Locations []string `json:"locations"`
}

type HoldingsFilterOptions struct {
	Brands    []string `json:"brands"`
	Locations []string `json:"locations"`
}

type SellUnitRequest struct {
	UnitID    string  `json:"unitId"`
	SoldAt    string  `json:"soldAt"`
	SalePrice float64 `json:"salePrice"`
}

type SellUnitsRequest struct {
	Units []SellUnitRequest `json:"units"`
}

type BulkUpdateHoldingUnitsRequest struct {
	UnitIDs         []string `json:"unitIds"`
	Dealer          string   `json:"dealer"`
	StorageLocation string   `json:"storageLocation"`
	Notes           string   `json:"notes"`
}

type AppSettings struct {
	DisplayCurrency  Currency      `json:"displayCurrency"`
	PriceSource      PriceSource   `json:"priceSource"`
	MetalpriceAPIKey string        `json:"metalpriceApiKey"`
	MiddlemanBaseURL string        `json:"middlemanBaseUrl"`
	AutoLockMinutes  int           `json:"autoLockMinutes"`
	SpotPriceUnit    SpotPriceUnit `json:"spotPriceUnit"`
	UITheme          UITheme       `json:"uiTheme"`
}

type PortfolioSummary struct {
	DisplayCurrency       Currency `json:"displayCurrency"`
	TotalPurchaseCost     float64  `json:"totalPurchaseCost"`
	TotalCurrentWorth     float64  `json:"totalCurrentWorth"`
	TotalRealizedProfit   float64  `json:"totalRealizedProfit"`
	TotalUnrealizedProfit float64  `json:"totalUnrealizedProfit"`
	TotalProfit           float64  `json:"totalProfit"`
	TotalProfitPct        float64  `json:"totalProfitPct"`
	HeldUnits             int      `json:"heldUnits"`
	SoldUnits             int      `json:"soldUnits"`
	GoldSpotPerKg         float64  `json:"goldSpotPerKg"`
	SilverSpotPerKg       float64  `json:"silverSpotPerKg"`
	QuoteAsOf             string   `json:"quoteAsOf"`
	QuoteIsStale          bool     `json:"quoteIsStale"`
	QuoteIsPartial        bool     `json:"quoteIsPartial"`
	QuoteCacheHit         bool     `json:"quoteCacheHit"`
	ValuationApproximate  bool     `json:"valuationApproximate"`
	PriceErrorCode        string   `json:"priceErrorCode,omitempty"`
}

type PortfolioHistoryPoint struct {
	Date           string  `json:"date"`
	PortfolioWorth float64 `json:"portfolioWorth"`
	CostBasis      float64 `json:"costBasis"`
	GoldWorth      float64 `json:"goldWorth"`
	GoldCostBasis  float64 `json:"goldCostBasis"`
	SilverWorth    float64 `json:"silverWorth"`
	SilverCostBasis float64 `json:"silverCostBasis"`
}

type PortfolioValueAt struct {
	Date                  string   `json:"date"`
	DisplayCurrency       Currency `json:"displayCurrency"`
	PortfolioWorth        float64  `json:"portfolioWorth"`
	CostBasis             float64  `json:"costBasis"`
	UnrealizedProfit      float64  `json:"unrealizedProfit"`
	UnrealizedProfitPct   float64  `json:"unrealizedProfitPct"`
	HeldUnits             int      `json:"heldUnits"`
	GoldWorth             float64  `json:"goldWorth"`
	SilverWorth           float64  `json:"silverWorth"`
	QuoteAsOf             string   `json:"quoteAsOf"`
	QuoteIsStale          bool     `json:"quoteIsStale"`
	QuoteIsPartial        bool     `json:"quoteIsPartial"`
	QuoteCacheHit         bool     `json:"quoteCacheHit"`
	ValuationApproximate  bool     `json:"valuationApproximate"`
	PriceErrorCode        string   `json:"priceErrorCode,omitempty"`
}

type MonthlyMetalBreakdown struct {
	YearMonth        string      `json:"yearMonth"`
	Metal            MetalSymbol `json:"metal"`
	OpeningWorth     float64     `json:"openingWorth"`
	UnrealizedChange float64     `json:"unrealizedChange"`
	UnrealizedPct    float64     `json:"unrealizedPct"`
	RealizedProfit   float64     `json:"realizedProfit"`
	RealizedPct      float64     `json:"realizedPct"`
	NetChange        float64     `json:"netChange"`
	NetPct           float64     `json:"netPct"`
}

type UpdateHoldingUnitRequest struct {
	UnitID              string      `json:"unitId"`
	Metal               MetalSymbol `json:"metal"`
	Form                Form        `json:"form"`
	Weight              float64     `json:"weight"`
	WeightUnit          WeightUnit  `json:"weightUnit"`
	Purity              float64     `json:"purity"`
	Brand               string      `json:"brand"`
	ProductName         string      `json:"productName"`
	PurchasePrice       float64     `json:"purchasePrice"`
	SpotWorthAtPurchase float64     `json:"spotWorthAtPurchase"`
	PurchasedAt         string      `json:"purchasedAt"`
	Notes               string      `json:"notes"`
	Dealer              string      `json:"dealer"`
	Status              UnitStatus  `json:"status"`
	SoldAt              string      `json:"soldAt"`
	SalePrice       *float64 `json:"salePrice"`
	StorageLocation string   `json:"storageLocation"`
	Condition       string   `json:"condition"`
	MintageYear     int      `json:"mintageYear"`
}

type SpotQuote struct {
	Base       string             `json:"base"`
	Timestamp  string             `json:"timestamp"`
	FetchedAt  string             `json:"fetchedAt"`
	Rates      map[string]float64 `json:"rates"`
	CacheHit   bool               `json:"cacheHit"`
	IsStale    bool               `json:"isStale"`
	IsPartial  bool               `json:"isPartial"`
	ErrorCode  string             `json:"errorCode,omitempty"`
}

type AllocationSlice struct {
	Key     string  `json:"key"`
	Label   string  `json:"label"`
	Worth   float64 `json:"worth"`
	Percent float64 `json:"percent"`
}

type AllocationBreakdown struct {
	DisplayCurrency Currency          `json:"displayCurrency"`
	TotalWorth      float64           `json:"totalWorth"`
	ByMetal         []AllocationSlice `json:"byMetal"`
	ByForm          []AllocationSlice `json:"byForm"`
	ByBrand         []AllocationSlice `json:"byBrand"`
	ByLocation      []AllocationSlice `json:"byLocation"`
}

type MetalAverageCost struct {
	Metal                  MetalSymbol `json:"metal"`
	TotalPurchaseCost      float64     `json:"totalPurchaseCost"`
	TotalFineWeightGrams   float64     `json:"totalFineWeightGrams"`
	AvgCostPerKgFine       float64     `json:"avgCostPerKgFine"`
	HeldUnits              int         `json:"heldUnits"`
}

type WhatIfRequest struct {
	GoldSpot   float64       `json:"goldSpot"`
	SilverSpot float64       `json:"silverSpot"`
	SpotUnit   SpotPriceUnit `json:"spotUnit"`
}

type WhatIfPreview struct {
	DisplayCurrency     Currency      `json:"displayCurrency"`
	SpotUnit            SpotPriceUnit `json:"spotUnit"`
	PortfolioWorth      float64       `json:"portfolioWorth"`
	BaselineWorth       float64       `json:"baselineWorth"`
	TotalPurchaseCost   float64       `json:"totalPurchaseCost"`
	UnrealizedProfit    float64       `json:"unrealizedProfit"`
	UnrealizedProfitPct float64       `json:"unrealizedProfitPct"`
	WorthDelta          float64       `json:"worthDelta"`
}

type PnLContribution struct {
	Dimension        string  `json:"dimension"`
	Key              string  `json:"key"`
	Label            string  `json:"label"`
	UnrealizedChange float64 `json:"unrealizedChange"`
	RealizedProfit   float64 `json:"realizedProfit"`
	NetChange        float64 `json:"netChange"`
	PercentOfTotalNet float64 `json:"percentOfTotalNet"`
}

type PnLContributionReport struct {
	DisplayCurrency  Currency          `json:"displayCurrency"`
	TotalUnrealized  float64           `json:"totalUnrealized"`
	TotalRealized    float64           `json:"totalRealized"`
	TotalNet         float64           `json:"totalNet"`
	ByMetal          []PnLContribution `json:"byMetal"`
	ByGroup          []PnLContribution `json:"byGroup"`
}

// MonthlyPage is the Monthly tab payload (one quote load for breakdown + contribution).
type MonthlyPage struct {
	Breakdown    []MonthlyMetalBreakdown `json:"breakdown"`
	Contribution PnLContributionReport   `json:"contribution"`
}

// UpdateCheckResult is returned by CheckForUpdates (GitHub Releases).
type UpdateCheckResult struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Available      bool   `json:"available"`
	ReleaseNotes   string `json:"releaseNotes"`
	AssetName      string `json:"assetName"`
}
