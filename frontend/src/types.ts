export type Currency = 'EUR' | 'USD' | 'CHF';
export type MetalSymbol = 'XAU' | 'XAG';
export type Form = 'coin' | 'bar' | 'other';
export type WeightUnit = 'g' | 'kg' | 'troy_oz';
export type PriceSource = 'metalpriceapi' | 'middleman';
export type SpotPriceUnit = 'g' | 'troy_oz' | 'kilogram';
export type UITheme = 'dark' | 'light';
export type UnitStatus = 'held' | 'sold';
export type AttachmentOwnerType = 'unit' | 'investment';
export type AttachmentKind = 'photo' | 'receipt';

export interface SetupResult {
  recoveryKey: string;
}

export interface BackupManifest {
  formatVersion: number;
  createdAt: string;
  unitCount: number;
  attachmentCount: number;
  checksums?: Record<string, string>;
}

export interface BackupVerifyResult {
  valid: boolean;
  formatVersion: number;
  createdAt: string;
  unitCount: number;
  attachmentCount: number;
  fileCount: number;
  message: string;
}

export interface AppSettings {
  displayCurrency: Currency;
  priceSource: PriceSource;
  metalpriceApiKey: string;
  middlemanBaseUrl: string;
  autoLockMinutes: number;
  spotPriceUnit: SpotPriceUnit;
  uiTheme: UITheme;
  skippedUpdateVersion: string;
}

export interface InvestmentLineInput {
  assetClass: string;
  metal: MetalSymbol;
  form: Form;
  weight: number;
  weightUnit: WeightUnit;
  purity: number;
  brand: string;
  productName: string;
  quantity: number;
  totalPurchasePrice: number;
  totalSpotWorth: number;
  spotWorthProvided: boolean;
  isGift: boolean;
  storageLocation: string;
  condition: string;
  mintageYear: number;
}

export interface CreateInvestmentRequest {
  purchasedAt: string;
  currency: Currency;
  notes: string;
  dealer: string;
  lines: InvestmentLineInput[];
}

export interface GroupedHolding {
  productKey: string;
  assetClass: string;
  metal: MetalSymbol;
  form: Form;
  weightGrams: number;
  purity: number;
  brand: string;
  productName: string;
  currency: Currency;
  displayCurrency: Currency;
  heldCount: number;
  soldCount: number;
  totalCount: number;
  totalWeightGrams: number;
  totalFineWeightGrams: number;
  totalPurchasePrice: number;
  totalCurrentWorth: number;
  totalProfit: number;
  totalProfitPct: number;
  totalRealizedProfit: number;
  totalUnrealizedProfit: number;
  valuationApproximate: boolean;
  breakEvenSpotPerKg?: number;
  avgCostPerKgFine?: number;
  heldPurchasePrice?: number;
  heldFineWeightGrams?: number;
}

export interface HoldingsFilter {
  search: string;
  metals: string[];
  forms: string[];
  brands: string[];
  locations: string[];
}

export interface HoldingsFilterOptions {
  brands: string[];
  locations: string[];
}

export interface Attachment {
  id: string;
  ownerType: AttachmentOwnerType | string;
  ownerId: string;
  kind: AttachmentKind | string;
  filename: string;
  contentType: string;
  relativePath: string;
  createdAt: string;
}

export interface AttachmentBytes {
  id: string;
  filename: string;
  contentType: string;
  dataBase64: string;
}

export interface DealerSummary {
  name: string;
  purchaseCount: number;
  unitCount: number;
  totalSpent: number;
  currency: Currency;
}

export interface UnitValuation {
  id: string;
  investmentId: string;
  assetClass: string;
  metal: MetalSymbol;
  form: Form;
  weightGrams: number;
  purity: number;
  brand: string;
  productName: string;
  productKey: string;
  currency: Currency;
  purchasePrice: number;
  spotWorthAtPurchase: number;
  isGift: boolean;
  purchasedAt: string;
  status: UnitStatus;
  soldAt?: string;
  salePrice?: number;
  notes: string;
  dealer: string;
  storageLocation: string;
  condition: string;
  mintageYear: number;
  deletedAt?: string;
  currentSpotWorth: number;
  premiumPaid: number;
  metalDelta: number;
  totalProfit: number;
  totalProfitPct: number;
  isRealized: boolean;
  fineWeightGrams: number;
  valuationApproximate: boolean;
  displayCurrency?: Currency;
  fxRateToDisplay?: number;
  breakEvenSpotPerKg?: number;
  daysHeld?: number;
  annualizedReturnPct?: number;
}

export interface SellUnitRequest {
  unitId: string;
  soldAt: string;
  salePrice: number;
}

export interface SellUnitsRequest {
  units: SellUnitRequest[];
}

export interface BulkUpdateHoldingUnitsRequest {
  unitIds: string[];
  dealer: string;
  storageLocation: string;
  notes: string;
}

export interface AddInvestmentPrefill {
  metal: MetalSymbol;
  form: Form;
  weightGrams: number;
  purity: number;
  brand: string;
  productName: string;
  condition?: string;
  mintageYear?: number;
}

export interface PortfolioSummary {
  displayCurrency: Currency;
  totalPurchaseCost: number;
  totalCurrentWorth: number;
  totalRealizedProfit: number;
  totalUnrealizedProfit: number;
  totalProfit: number;
  totalProfitPct: number;
  heldUnits: number;
  soldUnits: number;
  heldGoldFineWeightGrams: number;
  heldSilverFineWeightGrams: number;
  goldSpotPerKg: number;
  silverSpotPerKg: number;
  quoteAsOf: string;
  quoteIsStale: boolean;
  quoteIsPartial: boolean;
  quoteCacheHit: boolean;
  valuationApproximate: boolean;
  priceErrorCode?: string;
}

export interface PortfolioHistoryPoint {
  date: string;
  portfolioWorth: number;
  costBasis: number;
  goldWorth: number;
  goldCostBasis: number;
  silverWorth: number;
  silverCostBasis: number;
}

export interface PortfolioValueAt {
  date: string;
  displayCurrency: Currency;
  portfolioWorth: number;
  costBasis: number;
  unrealizedProfit: number;
  unrealizedProfitPct: number;
  heldUnits: number;
  goldWorth: number;
  silverWorth: number;
  quoteAsOf: string;
  quoteIsStale: boolean;
  quoteIsPartial: boolean;
  quoteCacheHit: boolean;
  valuationApproximate: boolean;
  priceErrorCode?: string;
}

export interface MonthlyMetalBreakdown {
  yearMonth: string;
  metal: MetalSymbol;
  openingWorth: number;
  unrealizedChange: number;
  unrealizedPct: number;
  realizedProfit: number;
  realizedPct: number;
  netChange: number;
  netPct: number;
}

export interface UpdateHoldingUnitRequest {
  unitId: string;
  metal: MetalSymbol;
  form: Form;
  weight: number;
  weightUnit: WeightUnit;
  purity: number;
  brand: string;
  productName: string;
  purchasePrice: number;
  spotWorthAtPurchase: number;
  isGift: boolean;
  purchasedAt: string;
  notes: string;
  dealer: string;
  status: UnitStatus;
  soldAt: string;
  salePrice?: number;
  storageLocation: string;
  condition: string;
  mintageYear: number;
}

export interface SpotQuote {
  base: string;
  timestamp: string;
  fetchedAt: string;
  rates: Record<string, number>;
  cacheHit: boolean;
  isStale: boolean;
  isPartial: boolean;
  errorCode?: string;
}

export interface AllocationSlice {
  key: string;
  label: string;
  worth: number;
  percent: number;
}

export interface AllocationBreakdown {
  displayCurrency: Currency;
  totalWorth: number;
  byMetal: AllocationSlice[];
  byForm: AllocationSlice[];
  byBrand: AllocationSlice[];
  byLocation: AllocationSlice[];
}

export interface MetalAverageCost {
  metal: MetalSymbol;
  totalPurchaseCost: number;
  totalFineWeightGrams: number;
  avgCostPerKgFine: number;
  heldUnits: number;
}

export interface WhatIfRequest {
  goldSpot: number;
  silverSpot: number;
  spotUnit: SpotPriceUnit;
}

export interface WhatIfPreview {
  displayCurrency: Currency;
  spotUnit: SpotPriceUnit;
  portfolioWorth: number;
  baselineWorth: number;
  totalPurchaseCost: number;
  unrealizedProfit: number;
  unrealizedProfitPct: number;
  worthDelta: number;
}

export interface PnLContribution {
  dimension: string;
  key: string;
  label: string;
  unrealizedChange: number;
  realizedProfit: number;
  netChange: number;
  percentOfTotalNet: number;
}

export interface PnLContributionReport {
  displayCurrency: Currency;
  totalUnrealized: number;
  totalRealized: number;
  totalNet: number;
  byMetal: PnLContribution[];
  byGroup: PnLContribution[];
}

export interface MonthlyPage {
  breakdown: MonthlyMetalBreakdown[];
  contribution: PnLContributionReport;
}

export interface UpdateCheckResult {
  currentVersion: string;
  latestVersion: string;
  available: boolean;
  releaseNotes: string;
  assetName: string;
}

export type AppView =
  | 'dashboard'
  | 'holdings'
  | 'sold'
  | 'add'
  | 'group'
  | 'unit'
  | 'monthly'
  | 'settings'
  | 'help';
