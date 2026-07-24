export namespace domain {
	
	export class AllocationSlice {
	    key: string;
	    label: string;
	    worth: number;
	    percent: number;
	
	    static createFrom(source: any = {}) {
	        return new AllocationSlice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.worth = source["worth"];
	        this.percent = source["percent"];
	    }
	}
	export class AllocationBreakdown {
	    displayCurrency: string;
	    totalWorth: number;
	    byMetal: AllocationSlice[];
	    byForm: AllocationSlice[];
	    byBrand: AllocationSlice[];
	    byLocation: AllocationSlice[];
	
	    static createFrom(source: any = {}) {
	        return new AllocationBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayCurrency = source["displayCurrency"];
	        this.totalWorth = source["totalWorth"];
	        this.byMetal = this.convertValues(source["byMetal"], AllocationSlice);
	        this.byForm = this.convertValues(source["byForm"], AllocationSlice);
	        this.byBrand = this.convertValues(source["byBrand"], AllocationSlice);
	        this.byLocation = this.convertValues(source["byLocation"], AllocationSlice);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class AppSettings {
	    displayCurrency: string;
	    priceSource: string;
	    metalpriceApiKey: string;
	    middlemanBaseUrl: string;
	    autoLockMinutes: number;
	    spotPriceUnit: string;
	    uiTheme: string;
	    skippedUpdateVersion: string;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayCurrency = source["displayCurrency"];
	        this.priceSource = source["priceSource"];
	        this.metalpriceApiKey = source["metalpriceApiKey"];
	        this.middlemanBaseUrl = source["middlemanBaseUrl"];
	        this.autoLockMinutes = source["autoLockMinutes"];
	        this.spotPriceUnit = source["spotPriceUnit"];
	        this.uiTheme = source["uiTheme"];
	        this.skippedUpdateVersion = source["skippedUpdateVersion"];
	    }
	}
	export class Attachment {
	    id: string;
	    ownerType: string;
	    ownerId: string;
	    kind: string;
	    filename: string;
	    contentType: string;
	    relativePath: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Attachment(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ownerType = source["ownerType"];
	        this.ownerId = source["ownerId"];
	        this.kind = source["kind"];
	        this.filename = source["filename"];
	        this.contentType = source["contentType"];
	        this.relativePath = source["relativePath"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class AttachmentBytes {
	    id: string;
	    filename: string;
	    contentType: string;
	    dataBase64: string;
	
	    static createFrom(source: any = {}) {
	        return new AttachmentBytes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.filename = source["filename"];
	        this.contentType = source["contentType"];
	        this.dataBase64 = source["dataBase64"];
	    }
	}
	export class BulkUpdateHoldingUnitsRequest {
	    unitIds: string[];
	    dealer: string;
	    storageLocation: string;
	    notes: string;
	
	    static createFrom(source: any = {}) {
	        return new BulkUpdateHoldingUnitsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unitIds = source["unitIds"];
	        this.dealer = source["dealer"];
	        this.storageLocation = source["storageLocation"];
	        this.notes = source["notes"];
	    }
	}
	export class InvestmentLineInput {
	    assetClass: string;
	    metal: string;
	    form: string;
	    weight: number;
	    weightUnit: string;
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
	
	    static createFrom(source: any = {}) {
	        return new InvestmentLineInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assetClass = source["assetClass"];
	        this.metal = source["metal"];
	        this.form = source["form"];
	        this.weight = source["weight"];
	        this.weightUnit = source["weightUnit"];
	        this.purity = source["purity"];
	        this.brand = source["brand"];
	        this.productName = source["productName"];
	        this.quantity = source["quantity"];
	        this.totalPurchasePrice = source["totalPurchasePrice"];
	        this.totalSpotWorth = source["totalSpotWorth"];
	        this.spotWorthProvided = source["spotWorthProvided"];
	        this.isGift = source["isGift"];
	        this.storageLocation = source["storageLocation"];
	        this.condition = source["condition"];
	        this.mintageYear = source["mintageYear"];
	    }
	}
	export class CreateInvestmentRequest {
	    purchasedAt: string;
	    currency: string;
	    notes: string;
	    dealer: string;
	    lines: InvestmentLineInput[];
	
	    static createFrom(source: any = {}) {
	        return new CreateInvestmentRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.purchasedAt = source["purchasedAt"];
	        this.currency = source["currency"];
	        this.notes = source["notes"];
	        this.dealer = source["dealer"];
	        this.lines = this.convertValues(source["lines"], InvestmentLineInput);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class DealerSummary {
	    name: string;
	    purchaseCount: number;
	    unitCount: number;
	    totalSpent: number;
	    currency: string;
	
	    static createFrom(source: any = {}) {
	        return new DealerSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.purchaseCount = source["purchaseCount"];
	        this.unitCount = source["unitCount"];
	        this.totalSpent = source["totalSpent"];
	        this.currency = source["currency"];
	    }
	}
	export class GroupedHolding {
	    productKey: string;
	    assetClass: string;
	    metal: string;
	    form: string;
	    weightGrams: number;
	    purity: number;
	    brand: string;
	    productName: string;
	    currency: string;
	    displayCurrency: string;
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
	
	    static createFrom(source: any = {}) {
	        return new GroupedHolding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.productKey = source["productKey"];
	        this.assetClass = source["assetClass"];
	        this.metal = source["metal"];
	        this.form = source["form"];
	        this.weightGrams = source["weightGrams"];
	        this.purity = source["purity"];
	        this.brand = source["brand"];
	        this.productName = source["productName"];
	        this.currency = source["currency"];
	        this.displayCurrency = source["displayCurrency"];
	        this.heldCount = source["heldCount"];
	        this.soldCount = source["soldCount"];
	        this.totalCount = source["totalCount"];
	        this.totalWeightGrams = source["totalWeightGrams"];
	        this.totalFineWeightGrams = source["totalFineWeightGrams"];
	        this.totalPurchasePrice = source["totalPurchasePrice"];
	        this.totalCurrentWorth = source["totalCurrentWorth"];
	        this.totalProfit = source["totalProfit"];
	        this.totalProfitPct = source["totalProfitPct"];
	        this.totalRealizedProfit = source["totalRealizedProfit"];
	        this.totalUnrealizedProfit = source["totalUnrealizedProfit"];
	        this.valuationApproximate = source["valuationApproximate"];
	        this.breakEvenSpotPerKg = source["breakEvenSpotPerKg"];
	        this.avgCostPerKgFine = source["avgCostPerKgFine"];
	        this.heldPurchasePrice = source["heldPurchasePrice"];
	        this.heldFineWeightGrams = source["heldFineWeightGrams"];
	    }
	}
	export class HoldingsFilter {
	    search: string;
	    metals: string[];
	    forms: string[];
	    brands: string[];
	    locations: string[];
	
	    static createFrom(source: any = {}) {
	        return new HoldingsFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.search = source["search"];
	        this.metals = source["metals"];
	        this.forms = source["forms"];
	        this.brands = source["brands"];
	        this.locations = source["locations"];
	    }
	}
	export class HoldingsFilterOptions {
	    brands: string[];
	    locations: string[];
	
	    static createFrom(source: any = {}) {
	        return new HoldingsFilterOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.brands = source["brands"];
	        this.locations = source["locations"];
	    }
	}
	
	export class MetalAverageCost {
	    metal: string;
	    totalPurchaseCost: number;
	    totalFineWeightGrams: number;
	    avgCostPerKgFine: number;
	    heldUnits: number;
	
	    static createFrom(source: any = {}) {
	        return new MetalAverageCost(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.metal = source["metal"];
	        this.totalPurchaseCost = source["totalPurchaseCost"];
	        this.totalFineWeightGrams = source["totalFineWeightGrams"];
	        this.avgCostPerKgFine = source["avgCostPerKgFine"];
	        this.heldUnits = source["heldUnits"];
	    }
	}
	export class MonthlyMetalBreakdown {
	    yearMonth: string;
	    metal: string;
	    openingWorth: number;
	    unrealizedChange: number;
	    unrealizedPct: number;
	    realizedProfit: number;
	    realizedPct: number;
	    netChange: number;
	    netPct: number;
	
	    static createFrom(source: any = {}) {
	        return new MonthlyMetalBreakdown(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.yearMonth = source["yearMonth"];
	        this.metal = source["metal"];
	        this.openingWorth = source["openingWorth"];
	        this.unrealizedChange = source["unrealizedChange"];
	        this.unrealizedPct = source["unrealizedPct"];
	        this.realizedProfit = source["realizedProfit"];
	        this.realizedPct = source["realizedPct"];
	        this.netChange = source["netChange"];
	        this.netPct = source["netPct"];
	    }
	}
	export class PnLContribution {
	    dimension: string;
	    key: string;
	    label: string;
	    unrealizedChange: number;
	    realizedProfit: number;
	    netChange: number;
	    percentOfTotalNet: number;
	
	    static createFrom(source: any = {}) {
	        return new PnLContribution(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dimension = source["dimension"];
	        this.key = source["key"];
	        this.label = source["label"];
	        this.unrealizedChange = source["unrealizedChange"];
	        this.realizedProfit = source["realizedProfit"];
	        this.netChange = source["netChange"];
	        this.percentOfTotalNet = source["percentOfTotalNet"];
	    }
	}
	export class PnLContributionReport {
	    displayCurrency: string;
	    totalUnrealized: number;
	    totalRealized: number;
	    totalNet: number;
	    byMetal: PnLContribution[];
	    byGroup: PnLContribution[];
	
	    static createFrom(source: any = {}) {
	        return new PnLContributionReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayCurrency = source["displayCurrency"];
	        this.totalUnrealized = source["totalUnrealized"];
	        this.totalRealized = source["totalRealized"];
	        this.totalNet = source["totalNet"];
	        this.byMetal = this.convertValues(source["byMetal"], PnLContribution);
	        this.byGroup = this.convertValues(source["byGroup"], PnLContribution);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MonthlyPage {
	    breakdown: MonthlyMetalBreakdown[];
	    contribution: PnLContributionReport;
	
	    static createFrom(source: any = {}) {
	        return new MonthlyPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.breakdown = this.convertValues(source["breakdown"], MonthlyMetalBreakdown);
	        this.contribution = this.convertValues(source["contribution"], PnLContributionReport);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class PortfolioHistoryPoint {
	    date: string;
	    portfolioWorth: number;
	    costBasis: number;
	    goldWorth: number;
	    goldCostBasis: number;
	    silverWorth: number;
	    silverCostBasis: number;
	
	    static createFrom(source: any = {}) {
	        return new PortfolioHistoryPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.portfolioWorth = source["portfolioWorth"];
	        this.costBasis = source["costBasis"];
	        this.goldWorth = source["goldWorth"];
	        this.goldCostBasis = source["goldCostBasis"];
	        this.silverWorth = source["silverWorth"];
	        this.silverCostBasis = source["silverCostBasis"];
	    }
	}
	export class PortfolioSummary {
	    displayCurrency: string;
	    totalPurchaseCost: number;
	    totalCurrentWorth: number;
	    totalRealizedProfit: number;
	    totalUnrealizedProfit: number;
	    totalProfit: number;
	    totalProfitPct: number;
	    heldUnits: number;
	    soldUnits: number;
	    goldSpotPerKg: number;
	    silverSpotPerKg: number;
	    quoteAsOf: string;
	    quoteIsStale: boolean;
	    quoteIsPartial: boolean;
	    quoteCacheHit: boolean;
	    valuationApproximate: boolean;
	    priceErrorCode?: string;
	
	    static createFrom(source: any = {}) {
	        return new PortfolioSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayCurrency = source["displayCurrency"];
	        this.totalPurchaseCost = source["totalPurchaseCost"];
	        this.totalCurrentWorth = source["totalCurrentWorth"];
	        this.totalRealizedProfit = source["totalRealizedProfit"];
	        this.totalUnrealizedProfit = source["totalUnrealizedProfit"];
	        this.totalProfit = source["totalProfit"];
	        this.totalProfitPct = source["totalProfitPct"];
	        this.heldUnits = source["heldUnits"];
	        this.soldUnits = source["soldUnits"];
	        this.goldSpotPerKg = source["goldSpotPerKg"];
	        this.silverSpotPerKg = source["silverSpotPerKg"];
	        this.quoteAsOf = source["quoteAsOf"];
	        this.quoteIsStale = source["quoteIsStale"];
	        this.quoteIsPartial = source["quoteIsPartial"];
	        this.quoteCacheHit = source["quoteCacheHit"];
	        this.valuationApproximate = source["valuationApproximate"];
	        this.priceErrorCode = source["priceErrorCode"];
	    }
	}
	export class PortfolioValueAt {
	    date: string;
	    displayCurrency: string;
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
	
	    static createFrom(source: any = {}) {
	        return new PortfolioValueAt(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.displayCurrency = source["displayCurrency"];
	        this.portfolioWorth = source["portfolioWorth"];
	        this.costBasis = source["costBasis"];
	        this.unrealizedProfit = source["unrealizedProfit"];
	        this.unrealizedProfitPct = source["unrealizedProfitPct"];
	        this.heldUnits = source["heldUnits"];
	        this.goldWorth = source["goldWorth"];
	        this.silverWorth = source["silverWorth"];
	        this.quoteAsOf = source["quoteAsOf"];
	        this.quoteIsStale = source["quoteIsStale"];
	        this.quoteIsPartial = source["quoteIsPartial"];
	        this.quoteCacheHit = source["quoteCacheHit"];
	        this.valuationApproximate = source["valuationApproximate"];
	        this.priceErrorCode = source["priceErrorCode"];
	    }
	}
	export class SellUnitRequest {
	    unitId: string;
	    soldAt: string;
	    salePrice: number;
	
	    static createFrom(source: any = {}) {
	        return new SellUnitRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unitId = source["unitId"];
	        this.soldAt = source["soldAt"];
	        this.salePrice = source["salePrice"];
	    }
	}
	export class SellUnitsRequest {
	    units: SellUnitRequest[];
	
	    static createFrom(source: any = {}) {
	        return new SellUnitsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.units = this.convertValues(source["units"], SellUnitRequest);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SpotQuote {
	    base: string;
	    timestamp: string;
	    fetchedAt: string;
	    rates: Record<string, number>;
	    cacheHit: boolean;
	    isStale: boolean;
	    isPartial: boolean;
	    errorCode?: string;
	
	    static createFrom(source: any = {}) {
	        return new SpotQuote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.base = source["base"];
	        this.timestamp = source["timestamp"];
	        this.fetchedAt = source["fetchedAt"];
	        this.rates = source["rates"];
	        this.cacheHit = source["cacheHit"];
	        this.isStale = source["isStale"];
	        this.isPartial = source["isPartial"];
	        this.errorCode = source["errorCode"];
	    }
	}
	export class UnitValuation {
	    id: string;
	    investmentId: string;
	    assetClass: string;
	    metal: string;
	    form: string;
	    weightGrams: number;
	    purity: number;
	    brand: string;
	    productName: string;
	    productKey: string;
	    currency: string;
	    purchasePrice: number;
	    spotWorthAtPurchase: number;
	    isGift: boolean;
	    purchasedAt: string;
	    status: string;
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
	    displayCurrency?: string;
	    fxRateToDisplay?: number;
	    breakEvenSpotPerKg?: number;
	    daysHeld?: number;
	    annualizedReturnPct?: number;
	
	    static createFrom(source: any = {}) {
	        return new UnitValuation(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.investmentId = source["investmentId"];
	        this.assetClass = source["assetClass"];
	        this.metal = source["metal"];
	        this.form = source["form"];
	        this.weightGrams = source["weightGrams"];
	        this.purity = source["purity"];
	        this.brand = source["brand"];
	        this.productName = source["productName"];
	        this.productKey = source["productKey"];
	        this.currency = source["currency"];
	        this.purchasePrice = source["purchasePrice"];
	        this.spotWorthAtPurchase = source["spotWorthAtPurchase"];
	        this.isGift = source["isGift"];
	        this.purchasedAt = source["purchasedAt"];
	        this.status = source["status"];
	        this.soldAt = source["soldAt"];
	        this.salePrice = source["salePrice"];
	        this.notes = source["notes"];
	        this.dealer = source["dealer"];
	        this.storageLocation = source["storageLocation"];
	        this.condition = source["condition"];
	        this.mintageYear = source["mintageYear"];
	        this.deletedAt = source["deletedAt"];
	        this.currentSpotWorth = source["currentSpotWorth"];
	        this.premiumPaid = source["premiumPaid"];
	        this.metalDelta = source["metalDelta"];
	        this.totalProfit = source["totalProfit"];
	        this.totalProfitPct = source["totalProfitPct"];
	        this.isRealized = source["isRealized"];
	        this.fineWeightGrams = source["fineWeightGrams"];
	        this.valuationApproximate = source["valuationApproximate"];
	        this.displayCurrency = source["displayCurrency"];
	        this.fxRateToDisplay = source["fxRateToDisplay"];
	        this.breakEvenSpotPerKg = source["breakEvenSpotPerKg"];
	        this.daysHeld = source["daysHeld"];
	        this.annualizedReturnPct = source["annualizedReturnPct"];
	    }
	}
	export class UpdateCheckResult {
	    currentVersion: string;
	    latestVersion: string;
	    available: boolean;
	    releaseNotes: string;
	    assetName: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.available = source["available"];
	        this.releaseNotes = source["releaseNotes"];
	        this.assetName = source["assetName"];
	    }
	}
	export class UpdateHoldingUnitRequest {
	    unitId: string;
	    metal: string;
	    form: string;
	    weight: number;
	    weightUnit: string;
	    purity: number;
	    brand: string;
	    productName: string;
	    purchasePrice: number;
	    spotWorthAtPurchase: number;
	    isGift: boolean;
	    purchasedAt: string;
	    notes: string;
	    dealer: string;
	    status: string;
	    soldAt: string;
	    salePrice?: number;
	    storageLocation: string;
	    condition: string;
	    mintageYear: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateHoldingUnitRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unitId = source["unitId"];
	        this.metal = source["metal"];
	        this.form = source["form"];
	        this.weight = source["weight"];
	        this.weightUnit = source["weightUnit"];
	        this.purity = source["purity"];
	        this.brand = source["brand"];
	        this.productName = source["productName"];
	        this.purchasePrice = source["purchasePrice"];
	        this.spotWorthAtPurchase = source["spotWorthAtPurchase"];
	        this.isGift = source["isGift"];
	        this.purchasedAt = source["purchasedAt"];
	        this.notes = source["notes"];
	        this.dealer = source["dealer"];
	        this.status = source["status"];
	        this.soldAt = source["soldAt"];
	        this.salePrice = source["salePrice"];
	        this.storageLocation = source["storageLocation"];
	        this.condition = source["condition"];
	        this.mintageYear = source["mintageYear"];
	    }
	}
	export class WhatIfPreview {
	    displayCurrency: string;
	    spotUnit: string;
	    portfolioWorth: number;
	    baselineWorth: number;
	    totalPurchaseCost: number;
	    unrealizedProfit: number;
	    unrealizedProfitPct: number;
	    worthDelta: number;
	
	    static createFrom(source: any = {}) {
	        return new WhatIfPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayCurrency = source["displayCurrency"];
	        this.spotUnit = source["spotUnit"];
	        this.portfolioWorth = source["portfolioWorth"];
	        this.baselineWorth = source["baselineWorth"];
	        this.totalPurchaseCost = source["totalPurchaseCost"];
	        this.unrealizedProfit = source["unrealizedProfit"];
	        this.unrealizedProfitPct = source["unrealizedProfitPct"];
	        this.worthDelta = source["worthDelta"];
	    }
	}
	export class WhatIfRequest {
	    goldSpot: number;
	    silverSpot: number;
	    spotUnit: string;
	
	    static createFrom(source: any = {}) {
	        return new WhatIfRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.goldSpot = source["goldSpot"];
	        this.silverSpot = source["silverSpot"];
	        this.spotUnit = source["spotUnit"];
	    }
	}

}

export namespace security {
	
	export class BackupManifest {
	    formatVersion: number;
	    createdAt: string;
	    unitCount: number;
	    attachmentCount: number;
	    checksums: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new BackupManifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.formatVersion = source["formatVersion"];
	        this.createdAt = source["createdAt"];
	        this.unitCount = source["unitCount"];
	        this.attachmentCount = source["attachmentCount"];
	        this.checksums = source["checksums"];
	    }
	}
	export class BackupVerifyResult {
	    valid: boolean;
	    formatVersion: number;
	    createdAt: string;
	    unitCount: number;
	    attachmentCount: number;
	    fileCount: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupVerifyResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.valid = source["valid"];
	        this.formatVersion = source["formatVersion"];
	        this.createdAt = source["createdAt"];
	        this.unitCount = source["unitCount"];
	        this.attachmentCount = source["attachmentCount"];
	        this.fileCount = source["fileCount"];
	        this.message = source["message"];
	    }
	}
	export class SetupResult {
	    recoveryKey: string;
	
	    static createFrom(source: any = {}) {
	        return new SetupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recoveryKey = source["recoveryKey"];
	    }
	}

}

