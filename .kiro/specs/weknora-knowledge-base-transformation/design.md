# aiplusall-kb 知识库改造技术规范 - 设计文档

## 概述

本设计文档详细描述了 aiplusall-kb 知识库改造项目的技术架构、数据模型、API 设计和实现方案。该项目将现有的单租户隔离系统升级为支持多层次权限模型的企业级知识管理平台，并针对法律领域进行专业化优化。

## 架构设计

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    aiplusall-kb 知识管理平台                      │
├─────────────────────────────────────────────────────────────┤
│  前端层 (Frontend)                                          │
│  ├── Web UI (React/Vue)                                    │
│  ├── Mobile App                                            │
│  └── API Gateway                                           │
├─────────────────────────────────────────────────────────────┤
│  API 层 (API Layer)                                        │
│  ├── 知识库管理 API                                         │
│  ├── 法律专业 API                                           │
│  ├── 分享管理 API                                           │
│  └── 图谱引擎 API                                           │
├─────────────────────────────────────────────────────────────┤
│  业务逻辑层 (Business Logic Layer)                          │
│  ├── 权限管理服务                                           │
│  ├── 知识库服务                                             │
│  ├── 法律专业服务                                           │
│  ├── 检索服务                                               │
│  └── 图谱引擎管理                                           │
├─────────────────────────────────────────────────────────────┤
│  数据访问层 (Data Access Layer)                             │
│  ├── 知识库仓储                                             │
│  ├── 分享仓储                                               │
│  ├── 审计仓储                                               │
│  └── 缓存管理                                               │
├─────────────────────────────────────────────────────────────┤
│  存储层 (Storage Layer)                                     │
│  ├── MySQL (元数据)                                         │
│  ├── Neo4j (知识图谱)                                       │
│  ├── Vector DB (向量存储)                                   │
│  ├── Redis (缓存)                                           │
│  └── 文件存储 (MinIO/S3)                                    │
└─────────────────────────────────────────────────────────────┘
```

### 核心组件设计

#### 1. 权限管理架构

```go
// 权限管理核心接口
type PermissionManager interface {
    CheckPermission(ctx context.Context, userID, resourceID string) (PermissionLevel, error)
    GrantPermission(ctx context.Context, share *KnowledgeBaseShare) error
    RevokePermission(ctx context.Context, shareID string) error
    ListUserPermissions(ctx context.Context, userID string) ([]*UserPermission, error)
}

// 权限级别定义
type PermissionLevel string
const (
    PermissionNone        PermissionLevel = "none"
    PermissionRead        PermissionLevel = "read"
    PermissionWrite       PermissionLevel = "write"
    PermissionAdmin       PermissionLevel = "admin"
    PermissionOwner       PermissionLevel = "owner"
    PermissionSystemAdmin PermissionLevel = "system_admin"
)
```

#### 2. 多引擎知识图谱架构

```go
// 知识图谱引擎管理器
type GraphEngineManager interface {
    RegisterEngine(engineType GraphEngineType, engine KnowledgeGraphEngine) error
    GetEngine(engineType GraphEngineType) (KnowledgeGraphEngine, error)
    SwitchEngine(ctx context.Context, kbID string, targetEngine GraphEngineType) error
    ListAvailableEngines() []EngineInfo
}

// 统一的知识图谱引擎接口
type KnowledgeGraphEngine interface {
    BuildGraph(ctx context.Context, chunks []*Chunk, config *GraphConfig) error
    QueryGraph(ctx context.Context, query string, params *QueryParams) (*GraphResult, error)
    RebuildGraph(ctx context.Context, kbID string, batchSize int) error
    GetEngineInfo() *EngineInfo
}
```

#### 4. 智能外部判例检索架构

```go
// 外部判例检索管理器
type ExternalCaseSearchManager interface {
    SearchCases(ctx context.Context, query *CaseSearchQuery) (*ExternalSearchResult, error)
    GetAvailableEngines() []ExternalSearchEngine
    ImportCase(ctx context.Context, caseData *ExternalCaseData, kbID string) (*types.Knowledge, error)
    ValidateSearchEngine(engineType ExternalSearchEngineType) error
}

// 外部搜索引擎接口 - 扩展现有的 WebSearchProvider
type ExternalSearchEngine interface {
    interfaces.WebSearchProvider // 继承现有接口
    SearchCases(ctx context.Context, query *CaseSearchQuery) (*RawSearchResult, error)
    ParseCaseData(ctx context.Context, rawData *RawSearchResult) ([]*ExternalCaseData, error)
    GetEngineInfo() *ExternalEngineInfo
    ValidateConfig() error
}

// 外部搜索引擎类型
type ExternalSearchEngineType string
const (
    EnginePerplexityLegal   ExternalSearchEngineType = "perplexity_legal"
    EngineBrowserUse        ExternalSearchEngineType = "browser_use"
    EngineCustomWeb         ExternalSearchEngineType = "custom_web"
    EngineCourtAPI          ExternalSearchEngineType = "court_api"
)
```

#### 3. 法律专业化架构

```go
// 法律文档分析器
type LegalDocumentAnalyzer interface {
    ExtractMetadata(ctx context.Context, content []byte) (*LegalDocumentMetadata, error)
    AnalyzeContract(ctx context.Context, contract *ContractDocument) (*ContractAnalysis, error)
    CheckCompliance(ctx context.Context, document *Document) (*ComplianceReport, error)
    FindSimilarCases(ctx context.Context, caseInfo *CaseInfo) ([]*SimilarCase, error)
}

// 法律检索增强器
type LegalSearchEnhancer interface {
    EnhanceQuery(ctx context.Context, query string, params *LegalSearchParams) (*EnhancedQuery, error)
    RankResults(ctx context.Context, results []*SearchResult, params *LegalSearchParams) ([]*SearchResult, error)
    ExtractLegalCitations(ctx context.Context, content string) ([]*LegalCitation, error)
}
```

## 数据模型设计

### 核心数据模型

#### 1. 扩展的知识库模型

```go
type KnowledgeBase struct {
    // 基础字段
    ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    Name        string    `gorm:"type:varchar(255);not null" json:"name"`
    Type        string    `gorm:"type:varchar(32);default:'document'" json:"type"`
    Description string    `gorm:"type:text" json:"description"`
    IsTemporary bool      `gorm:"default:false" json:"is_temporary"`
    
    // 租户和所有权
    TenantID      uint64  `gorm:"index;comment:资源租户ID" json:"tenant_id"`
    OwnerType     string  `gorm:"type:varchar(16);default:'tenant';index" json:"owner_type"`
    OwnerTenantID uint64  `gorm:"index" json:"owner_tenant_id"`
    OwnerUserID   *string `gorm:"type:varchar(36);index" json:"owner_user_id"`
    
    // 可见性和领域
    Visibility string `gorm:"type:varchar(16);default:'private';index" json:"visibility"`
    Domain     string `gorm:"type:varchar(32);index" json:"domain"`
    
    // 配置
    ChunkingConfig            *ChunkingConfig            `gorm:"type:json" json:"chunking_config"`
    VLMConfig                *VLMConfig                 `gorm:"type:json" json:"vlm_config"`
    ExtractConfig            *ExtractConfig             `gorm:"type:json" json:"extract_config"`
    FAQConfig                *FAQConfig                 `gorm:"type:json" json:"faq_config"`
    QuestionGenerationConfig *QuestionGenerationConfig `gorm:"type:json" json:"question_generation_config"`
    
    // 模型配置
    EmbeddingModelID string `gorm:"type:varchar(36)" json:"embedding_model_id"`
    SummaryModelID   string `gorm:"type:varchar(36)" json:"summary_model_id"`
    
    // 统计信息
    KnowledgeCount int64 `gorm:"default:0" json:"knowledge_count"`
    ChunkCount     int64 `gorm:"default:0" json:"chunk_count"`
    StorageUsed    int64 `gorm:"default:0" json:"storage_used"`
    
    // 时间戳
    CreatedAt time.Time `gorm:"index" json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

#### 2. 增强的知识文档模型

```go
type Knowledge struct {
    // 基础字段
    ID              string `gorm:"type:varchar(36);primaryKey" json:"id"`
    TenantID        uint64 `gorm:"index" json:"tenant_id"`
    KnowledgeBaseID string `gorm:"type:varchar(36);index" json:"knowledge_base_id"`
    TagID           string `gorm:"type:varchar(36);index" json:"tag_id"`
    
    // 内容字段
    Title       string `gorm:"type:varchar(255)" json:"title"`
    Description string `gorm:"type:text" json:"description"`
    Source      string `gorm:"type:varchar(255)" json:"source"`
    FileHash    string `gorm:"type:varchar(64);index" json:"file_hash"`
    Type        string `gorm:"type:varchar(32);default:'manual'" json:"type"`
    
    // 状态字段
    ParseStatus   string `gorm:"type:varchar(32);default:'pending'" json:"parse_status"`
    SummaryStatus string `gorm:"type:varchar(32);default:'pending'" json:"summary_status"`
    EnableStatus  bool   `gorm:"default:true" json:"enable_status"`
    
    // 法律专业元数据
    LegalMetadata *LegalDocumentMetadata `gorm:"type:json" json:"legal_metadata"`
    
    // 统计信息
    ChunkCount int `gorm:"default:0" json:"chunk_count"`
    ViewCount  int `gorm:"default:0" json:"view_count"`
    
    // 时间戳
    CreatedAt time.Time `gorm:"index" json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

// 法律文档元数据
type LegalDocumentMetadata struct {
    // 文档分类
    DocumentType  LegalDocumentType `json:"document_type"`
    LegalDomain   string           `json:"legal_domain"`
    Jurisdiction  string           `json:"jurisdiction"`
    
    // 时效性
    EffectiveDate *time.Time `json:"effective_date"`
    ExpiryDate    *time.Time `json:"expiry_date"`
    LastUpdated   *time.Time `json:"last_updated"`
    
    // 权威性
    LegalAuthority string `json:"legal_authority"`
    AuthorityLevel string `json:"authority_level"`
    CitationFormat string `json:"citation_format"`
    
    // 内容特征
    KeyConcepts     []string `json:"key_concepts"`
    RelatedStatutes []string `json:"related_statutes"`
    CaseNumbers     []string `json:"case_numbers"`
    
    // 合规与风险
    SensitivityLevel string   `json:"sensitivity_level"`
    ComplianceFlags  []string `json:"compliance_flags"`
    RiskLevel        string   `json:"risk_level"`
}

// 法律文档类型
type LegalDocumentType string
const (
    Statute        LegalDocumentType = "statute"
    CaseLaw        LegalDocumentType = "case_law"
    Contract       LegalDocumentType = "contract"
    LegalOpinion   LegalDocumentType = "legal_opinion"
    Regulation     LegalDocumentType = "regulation"
    Interpretation LegalDocumentType = "interpretation"
    Precedent      LegalDocumentType = "precedent"
    LegalBrief     LegalDocumentType = "legal_brief"
    Memorandum     LegalDocumentType = "memorandum"
)
```

#### 3. 知识库分享模型

```go
type KnowledgeBaseShare struct {
    ID              string     `gorm:"type:varchar(36);primaryKey" json:"id"`
    KnowledgeBaseID string     `gorm:"type:varchar(36);index" json:"knowledge_base_id"`
    SharedByUserID  string     `gorm:"type:varchar(36);index" json:"shared_by_user_id"`
    
    // 分享目标
    TargetType     string  `gorm:"type:varchar(16);index" json:"target_type"`
    TargetUserID   *string `gorm:"type:varchar(36);index" json:"target_user_id"`
    TargetTenantID *uint64 `gorm:"index" json:"target_tenant_id"`
    
    // 权限和状态
    Permission string     `gorm:"type:varchar(16);default:'read';index" json:"permission"`
    ExpiresAt  *time.Time `gorm:"index" json:"expires_at"`
    IsEnabled  bool       `gorm:"default:true;index" json:"is_enabled"`
    
    // 元数据
    ShareNote string `gorm:"type:text" json:"share_note"`
    ShareURL  string `gorm:"type:varchar(255)" json:"share_url"`
    
    // 时间戳
    CreatedAt time.Time `gorm:"index" json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

#### 4. 法律合规审计日志

```go
type LegalComplianceAuditLog struct {
    ID              string    `gorm:"type:varchar(36);primaryKey" json:"id"`
    KnowledgeBaseID string    `gorm:"type:varchar(36);index" json:"knowledge_base_id"`
    KnowledgeID     *string   `gorm:"type:varchar(36);index" json:"knowledge_id"`
    UserID          string    `gorm:"type:varchar(36);index" json:"user_id"`
    TenantID        uint64    `gorm:"index" json:"tenant_id"`
    
    // 操作信息
    Action          string `gorm:"type:varchar(32);index" json:"action"`
    ActionDetails   string `gorm:"type:text" json:"action_details"`
    ResourcePath    string `gorm:"type:varchar(255)" json:"resource_path"`
    
    // 法律合规字段
    LegalDomain       string `gorm:"type:varchar(64);index" json:"legal_domain"`
    Jurisdiction      string `gorm:"type:varchar(64);index" json:"jurisdiction"`
    SensitivityLevel  string `gorm:"type:varchar(16);index" json:"sensitivity_level"`
    ComplianceStatus  string `gorm:"type:varchar(16);index" json:"compliance_status"`
    DataClassification string `gorm:"type:varchar(32);index" json:"data_classification"`
    
    // 技术信息
    IPAddress    string `gorm:"type:varchar(45)" json:"ip_address"`
    UserAgent    string `gorm:"type:text" json:"user_agent"`
    SessionID    string `gorm:"type:varchar(64);index" json:"session_id"`
    RequestID    string `gorm:"type:varchar(64);index" json:"request_id"`
    ResponseCode int    `gorm:"index" json:"response_code"`
    
    // 时间戳
    CreatedAt time.Time `gorm:"index" json:"created_at"`
}
```

#### 4. 外部判例数据模型

```go
// 外部案例搜索查询
type CaseSearchQuery struct {
    // 基础查询信息
    Query          string   `json:"query"`
    CaseType       string   `json:"case_type"`       // 案件类型
    Court          string   `json:"court"`           // 审理法院
    Jurisdiction   string   `json:"jurisdiction"`    // 司法管辖区
    DateRange      *DateRange `json:"date_range"`    // 时间范围
    
    // 案件要素
    Parties        []string `json:"parties"`         // 当事人
    CauseOfAction  string   `json:"cause_of_action"` // 案由
    LegalIssues    []string `json:"legal_issues"`    // 争议焦点
    
    // 搜索配置
    MaxResults     int      `json:"max_results"`
    EngineType     ExternalSearchEngineType `json:"engine_type"`
    EnableCache    bool     `json:"enable_cache"`
    Timeout        int      `json:"timeout"`
}

// 外部案例数据
type ExternalCaseData struct {
    // 基础信息
    CaseNumber     string    `json:"case_number"`     // 案件编号
    CaseTitle      string    `json:"case_title"`      // 案件标题
    Court          string    `json:"court"`           // 审理法院
    JudgeDate      time.Time `json:"judge_date"`      // 判决日期
    CaseType       string    `json:"case_type"`       // 案件类型
    
    // 当事人信息
    Plaintiff      []string  `json:"plaintiff"`       // 原告
    Defendant      []string  `json:"defendant"`       // 被告
    ThirdParty     []string  `json:"third_party"`     // 第三人
    
    // 案件内容
    CauseOfAction  string    `json:"cause_of_action"` // 案由
    CaseFacts      string    `json:"case_facts"`      // 案件事实
    LegalIssues    []string  `json:"legal_issues"`    // 争议焦点
    CourtOpinion   string    `json:"court_opinion"`   // 法院观点
    JudgmentResult string    `json:"judgment_result"` // 判决结果
    
    // 法律依据
    AppliedLaws    []string  `json:"applied_laws"`    // 适用法条
    LegalPrinciple string    `json:"legal_principle"` // 法律原则
    
    // 元数据
    SourceURL      string    `json:"source_url"`      // 来源链接
    Confidence     float64   `json:"confidence"`      // 相关性评分
    ExtractedAt    time.Time `json:"extracted_at"`    // 提取时间
    
    // 相似度信息
    SimilarityScore float64  `json:"similarity_score"` // 与查询的相似度
    MatchedElements []string `json:"matched_elements"` // 匹配的要素
}

// 外部搜索结果
type ExternalSearchResult struct {
    Query          *CaseSearchQuery   `json:"query"`
    Cases          []*ExternalCaseData `json:"cases"`
    TotalFound     int               `json:"total_found"`
    SearchTime     time.Duration     `json:"search_time"`
    EngineUsed     ExternalSearchEngineType `json:"engine_used"`
    CacheHit       bool              `json:"cache_hit"`
    NextPageToken  string            `json:"next_page_token"`
}
```

### 数据库索引设计

```sql
-- 知识库表索引
CREATE INDEX idx_kb_owner_type ON knowledge_bases(owner_type);
CREATE INDEX idx_kb_owner_tenant_id ON knowledge_bases(owner_tenant_id);
CREATE INDEX idx_kb_owner_user_id ON knowledge_bases(owner_user_id);
CREATE INDEX idx_kb_visibility ON knowledge_bases(visibility);
CREATE INDEX idx_kb_domain ON knowledge_bases(domain);
CREATE INDEX idx_kb_composite ON knowledge_bases(tenant_id, owner_type, visibility);

-- 知识文档表索引
CREATE INDEX idx_knowledge_legal_doc_type ON knowledge((legal_metadata->>'$.document_type'));
CREATE INDEX idx_knowledge_legal_domain ON knowledge((legal_metadata->>'$.legal_domain'));
CREATE INDEX idx_knowledge_jurisdiction ON knowledge((legal_metadata->>'$.jurisdiction'));
CREATE INDEX idx_knowledge_effective_date ON knowledge((legal_metadata->>'$.effective_date'));
CREATE INDEX idx_knowledge_authority_level ON knowledge((legal_metadata->>'$.authority_level'));
CREATE INDEX idx_knowledge_risk_level ON knowledge((legal_metadata->>'$.risk_level'));

-- 分享表索引
CREATE INDEX idx_share_target_user ON knowledge_base_shares(target_type, target_user_id);
CREATE INDEX idx_share_target_tenant ON knowledge_base_shares(target_type, target_tenant_id);
CREATE INDEX idx_share_expires ON knowledge_base_shares(expires_at, is_enabled);
CREATE INDEX idx_share_permission ON knowledge_base_shares(permission, is_enabled);

-- 审计日志索引
CREATE INDEX idx_audit_user_action ON legal_compliance_audit_logs(user_id, action, created_at);
CREATE INDEX idx_audit_legal_domain ON legal_compliance_audit_logs(legal_domain, created_at);
CREATE INDEX idx_audit_compliance ON legal_compliance_audit_logs(compliance_status, sensitivity_level);

-- 外部搜索缓存表
CREATE TABLE external_search_cache (
    id VARCHAR(36) PRIMARY KEY,
    query_hash VARCHAR(64) UNIQUE NOT NULL,
    query_text TEXT NOT NULL,
    search_params JSON,
    result_data JSON,
    engine_type VARCHAR(32) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    hit_count INT DEFAULT 0,
    
    INDEX idx_cache_query_hash (query_hash),
    INDEX idx_cache_engine_type (engine_type),
    INDEX idx_cache_expires_at (expires_at)
);
```

## API 设计

### 1. 知识库管理 API

#### 创建用户私有知识库
```http
POST /api/v1/my/knowledge-bases
Content-Type: application/json

{
  "name": "个人法律文档库",
  "type": "document",
  "description": "个人收集的法律文档",
  "domain": "legal",
  "visibility": "private"
}

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "id": "kb-xxx",
    "name": "个人法律文档库",
    "owner_type": "user",
    "visibility": "private",
    "domain": "legal",
    "my_permission": "owner",
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### 获取可访问知识库列表
```http
GET /api/v1/knowledge-bases?scope=all&include_shared=true&domain=legal

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "items": [
      {
        "id": "kb-xxx",
        "name": "企业法律知识库",
        "owner_type": "tenant",
        "visibility": "private",
        "domain": "legal",
        "my_permission": "read",
        "knowledge_count": 1500,
        "created_at": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 1,
    "page": 1,
    "page_size": 20
  }
}
```

### 2. 分享管理 API

#### 创建分享
```http
POST /api/v1/knowledge-bases/kb-xxx/shares
Content-Type: application/json

{
  "shares": [
    {
      "target_type": "user",
      "target_user_id": "user-456",
      "permission": "read",
      "expires_at": "2025-12-31T23:59:59Z",
      "share_note": "分享法律案例库供参考"
    }
  ]
}

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "created_shares": [
      {
        "id": "share-xxx",
        "target_user_id": "user-456",
        "permission": "read",
        "expires_at": "2025-12-31T23:59:59Z",
        "share_url": "https://aiplusall-kb.com/shared/kb-xxx?token=xxx"
      }
    ]
  }
}
```

### 3. 法律专业 API

#### 法律文档检索
```http
POST /api/v1/legal/search
Content-Type: application/json

{
  "query": "合同违约责任",
  "knowledge_base_ids": ["kb-xxx"],
  "legal_params": {
    "document_types": ["statute", "case_law"],
    "legal_domains": ["contract_law"],
    "jurisdictions": ["china"],
    "authority_levels": ["supreme_court", "high_court"],
    "effective_date_range": {
      "start": "2020-01-01",
      "end": "2024-12-31"
    },
    "include_expired": false
  },
  "top_k": 10
}

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "results": [
      {
        "knowledge_id": "know-xxx",
        "title": "合同法第107条 - 违约责任",
        "content_snippet": "当事人一方不履行合同义务或者履行合同义务不符合约定的，应当承担继续履行、采取补救措施或者赔偿损失等违约责任。",
        "legal_metadata": {
          "document_type": "statute",
          "legal_domain": "contract_law",
          "jurisdiction": "china",
          "authority_level": "national_law",
          "citation_format": "《中华人民共和国合同法》第107条"
        },
        "relevance_score": 0.95,
        "legal_relevance_score": 0.98
      }
    ],
    "total": 1,
    "search_time_ms": 150
  }
}
```

#### 合同风险分析
```http
POST /api/v1/legal/contract/analyze
Content-Type: application/json

{
  "knowledge_id": "know-xxx",
  "analysis_type": "risk_assessment"
}

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "overall_risk_level": "medium",
    "risk_points": [
      {
        "category": "liability_clause",
        "risk_level": "high",
        "description": "违约责任条款不明确",
        "suggestion": "建议明确违约责任的具体承担方式和赔偿标准",
        "clause_location": "第8条第2款"
      }
    ],
    "compliance_check": {
      "status": "passed",
      "checked_regulations": ["contract_law", "civil_code"],
      "compliance_score": 0.85
    },
    "confidence": 0.92
  }
}
```

### 4. 外部判例检索 API

#### 智能判例搜索
```http
POST /api/v1/legal/cases/search-external
Content-Type: application/json

{
  "query": "网络购物合同纠纷退货责任",
  "case_type": "civil",
  "jurisdiction": "china",
  "date_range": {
    "start": "2020-01-01",
    "end": "2024-12-31"
  },
  "legal_issues": ["合同履行", "消费者权益保护"],
  "engine_preferences": ["perplexity", "browser_use"],
  "max_results": 10,
  "enable_auto_import": false
}

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "local_results": [
      {
        "knowledge_id": "know-xxx",
        "title": "某电商平台退货纠纷案",
        "similarity_score": 0.89,
        "source": "local"
      }
    ],
    "external_results": [
      {
        "case_number": "(2023)京0105民初12345号",
        "case_title": "张某诉某电商公司网络购物合同纠纷案",
        "court": "北京市朝阳区人民法院",
        "judge_date": "2023-08-15T00:00:00Z",
        "case_facts": "原告张某在被告电商平台购买商品后要求退货...",
        "legal_issues": ["合同履行义务", "消费者权益保护"],
        "judgment_result": "支持原告退货请求，被告承担退货运费",
        "similarity_score": 0.92,
        "source_url": "https://wenshu.court.gov.cn/...",
        "confidence": 0.95,
        "matched_elements": ["网络购物", "退货责任", "消费者权益"]
      }
    ],
    "search_summary": {
      "total_local": 1,
      "total_external": 5,
      "search_time_ms": 3500,
      "engines_used": ["perplexity", "browser_use"],
      "cache_hit": false
    }
  }
}
```

#### 导入外部判例
```http
POST /api/v1/legal/cases/import-external
Content-Type: application/json

{
  "knowledge_base_id": "kb-xxx",
  "external_case": {
    "case_number": "(2023)京0105民初12345号",
    "case_title": "张某诉某电商公司网络购物合同纠纷案",
    "source_url": "https://wenshu.court.gov.cn/...",
    "case_facts": "...",
    "judgment_result": "..."
  },
  "import_options": {
    "auto_extract_metadata": true,
    "enable_graph_analysis": true,
    "set_as_reference": true
  }
}

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "knowledge_id": "know-new-xxx",
    "title": "张某诉某电商公司网络购物合同纠纷案",
    "import_status": "completed",
    "extracted_metadata": {
      "document_type": "case_law",
      "legal_domain": "contract_law",
      "key_concepts": ["网络购物", "合同纠纷", "退货责任"]
    }
  }
}
```

### 5. 图谱引擎管理 API

#### 获取可用引擎列表
```http
GET /api/v1/graph-engines

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "engines": [
      {
        "type": "aiplusall-kb",
        "name": "aiplusall-kb Native",
        "description": "aiplusall-kb原生知识图谱引擎",
        "features": ["Neo4j存储", "实体关系抽取", "异步重建"],
        "enabled": true,
        "recommended_for": ["快速响应", "现有集成"]
      },
      {
        "type": "graphrag",
        "name": "Microsoft GraphRAG",
        "description": "Microsoft GraphRAG知识图谱引擎",
        "features": ["社区检测", "全局查询", "层次化分析"],
        "enabled": true,
        "recommended_for": ["大规模分析", "抽象问题"]
      }
    ]
  }
}
```

#### 切换知识库图谱引擎
```http
POST /api/v1/knowledge-bases/kb-xxx/graph-engine/switch
Content-Type: application/json

{
  "engine_type": "graphrag",
  "config": {
    "model_config": {
      "api_key": "${OPENAI_API_KEY}",
      "model_name": "gpt-4o-mini"
    },
    "engine_specific": {
      "community_detection": true,
      "max_gleanings": 1
    }
  }
}

Response:
{
  "code": 200,
  "message": "success",
  "data": {
    "task_id": "task-xxx",
    "message": "图谱引擎切换已开始",
    "estimated_time": "30分钟"
  }
}
```

## 组件实现

### 1. 权限管理服务

```go
// internal/application/service/knowledge_base_permission_service.go
package service

import (
    "context"
    "fmt"
    "time"
    
    "github.com/Tencent/aiplusall-kb/internal/types"
    "github.com/Tencent/aiplusall-kb/internal/types/interfaces"
    "github.com/Tencent/aiplusall-kb/internal/logger"
)

type KnowledgeBasePermissionService struct {
    kbRepo      interfaces.KnowledgeBaseRepository
    shareRepo   interfaces.KnowledgeBaseShareRepository
    userRepo    interfaces.UserRepository
    cacheRepo   interfaces.CacheRepository
}

func NewKnowledgeBasePermissionService(
    kbRepo interfaces.KnowledgeBaseRepository,
    shareRepo interfaces.KnowledgeBaseShareRepository,
    userRepo interfaces.UserRepository,
    cacheRepo interfaces.CacheRepository,
) *KnowledgeBasePermissionService {
    return &KnowledgeBasePermissionService{
        kbRepo:    kbRepo,
        shareRepo: shareRepo,
        userRepo:  userRepo,
        cacheRepo: cacheRepo,
    }
}

func (s *KnowledgeBasePermissionService) CheckPermission(
    ctx context.Context,
    userID string,
    kbID string,
) (types.PermissionLevel, error) {
    // 1. 检查缓存
    cacheKey := fmt.Sprintf("permission:%s:%s", userID, kbID)
    if cached, err := s.cacheRepo.Get(ctx, cacheKey); err == nil {
        return types.PermissionLevel(cached), nil
    }
    
    // 2. 获取用户信息
    user, err := s.userRepo.GetByID(ctx, userID)
    if err != nil {
        return types.PermissionNone, err
    }
    
    // 3. 超级管理员检查
    if user.CanAccessAllTenants {
        s.cachePermission(ctx, cacheKey, types.PermissionSystemAdmin)
        return types.PermissionSystemAdmin, nil
    }
    
    // 4. 获取知识库信息
    kb, err := s.kbRepo.GetByID(ctx, kbID)
    if err != nil {
        return types.PermissionNone, err
    }
    
    // 5. 所有者检查
    if s.isOwner(user, kb) {
        s.cachePermission(ctx, cacheKey, types.PermissionOwner)
        return types.PermissionOwner, nil
    }
    
    // 6. 分享权限检查
    sharePermission, err := s.getSharePermission(ctx, userID, user.TenantID, kbID)
    if err == nil && sharePermission != types.PermissionNone {
        s.cachePermission(ctx, cacheKey, sharePermission)
        return sharePermission, nil
    }
    
    // 7. 公开知识库检查
    if kb.Visibility == "public" {
        s.cachePermission(ctx, cacheKey, types.PermissionRead)
        return types.PermissionRead, nil
    }
    
    return types.PermissionNone, nil
}

func (s *KnowledgeBasePermissionService) isOwner(user *types.User, kb *types.KnowledgeBase) bool {
    switch kb.OwnerType {
    case "system":
        return user.CanAccessAllTenants
    case "tenant":
        return user.TenantID == kb.OwnerTenantID && user.Role == "admin"
    case "user":
        return kb.OwnerUserID != nil && user.ID == *kb.OwnerUserID
    default:
        return false
    }
}

func (s *KnowledgeBasePermissionService) getSharePermission(
    ctx context.Context,
    userID string,
    tenantID uint64,
    kbID string,
) (types.PermissionLevel, error) {
    shares, err := s.shareRepo.GetActiveSharesByKnowledgeBaseID(ctx, kbID)
    if err != nil {
        return types.PermissionNone, err
    }
    
    now := time.Now()
    for _, share := range shares {
        // 检查是否过期和启用
        if !share.IsEnabled || (share.ExpiresAt != nil && share.ExpiresAt.Before(now)) {
            continue
        }
        
        // 检查目标匹配
        if share.TargetType == "user" && share.TargetUserID != nil && *share.TargetUserID == userID {
            return types.PermissionLevel(share.Permission), nil
        }
        if share.TargetType == "tenant" && share.TargetTenantID != nil && *share.TargetTenantID == tenantID {
            return types.PermissionLevel(share.Permission), nil
        }
    }
    
    return types.PermissionNone, nil
}

func (s *KnowledgeBasePermissionService) cachePermission(
    ctx context.Context,
    cacheKey string,
    permission types.PermissionLevel,
) {
    // 缓存5分钟
    err := s.cacheRepo.Set(ctx, cacheKey, string(permission), 5*time.Minute)
    if err != nil {
        logger.Warnf(ctx, "Failed to cache permission: %v", err)
    }
}

func (s *KnowledgeBasePermissionService) InvalidatePermissionCache(
    ctx context.Context,
    userID string,
    kbID string,
) error {
    cacheKey := fmt.Sprintf("permission:%s:%s", userID, kbID)
    return s.cacheRepo.Delete(ctx, cacheKey)
}
```

### 2. 法律文档分析服务

```go
// internal/application/service/legal_document_analyzer.go
package service

import (
    "context"
    "encoding/json"
    "regexp"
    "strings"
    "time"
    
    "github.com/Tencent/aiplusall-kb/internal/types"
    "github.com/Tencent/aiplusall-kb/internal/types/interfaces"
    "github.com/Tencent/aiplusall-kb/internal/logger"
)

type LegalDocumentAnalyzer struct {
    nlpService    interfaces.NLPService
    modelService  interfaces.ModelService
    legalRuleRepo interfaces.LegalRuleRepository
}

func NewLegalDocumentAnalyzer(
    nlpService interfaces.NLPService,
    modelService interfaces.ModelService,
    legalRuleRepo interfaces.LegalRuleRepository,
) *LegalDocumentAnalyzer {
    return &LegalDocumentAnalyzer{
        nlpService:    nlpService,
        modelService:  modelService,
        legalRuleRepo: legalRuleRepo,
    }
}

func (s *LegalDocumentAnalyzer) ExtractMetadata(
    ctx context.Context,
    content []byte,
) (*types.LegalDocumentMetadata, error) {
    contentStr := string(content)
    
    // 1. 文档类型识别
    docType := s.identifyDocumentType(contentStr)
    
    // 2. 法律领域识别
    legalDomain := s.identifyLegalDomain(ctx, contentStr)
    
    // 3. 司法管辖区识别
    jurisdiction := s.identifyJurisdiction(contentStr)
    
    // 4. 提取关键概念
    keyConcepts, err := s.extractKeyConcepts(ctx, contentStr)
    if err != nil {
        logger.Warnf(ctx, "Failed to extract key concepts: %v", err)
        keyConcepts = []string{}
    }
    
    // 5. 提取法条引用
    relatedStatutes := s.extractStatuteReferences(contentStr)
    
    // 6. 风险评估
    riskLevel := s.assessRiskLevel(ctx, contentStr, docType)
    
    // 7. 合规检查
    complianceFlags := s.checkCompliance(ctx, contentStr, docType)
    
    metadata := &types.LegalDocumentMetadata{
        DocumentType:     docType,
        LegalDomain:      legalDomain,
        Jurisdiction:     jurisdiction,
        KeyConcepts:      keyConcepts,
        RelatedStatutes:  relatedStatutes,
        RiskLevel:        riskLevel,
        ComplianceFlags:  complianceFlags,
        SensitivityLevel: s.assessSensitivityLevel(docType, riskLevel),
    }
    
    // 8. 提取时效性信息
    s.extractTemporalInfo(contentStr, metadata)
    
    // 9. 提取权威性信息
    s.extractAuthorityInfo(contentStr, metadata)
    
    return metadata, nil
}

func (s *LegalDocumentAnalyzer) identifyDocumentType(content string) types.LegalDocumentType {
    content = strings.ToLower(content)
    
    // 法律法规关键词
    if s.containsAny(content, []string{"法律", "条例", "规定", "办法", "细则"}) {
        return types.Statute
    }
    
    // 判例关键词
    if s.containsAny(content, []string{"判决书", "裁定书", "案例", "判例"}) {
        return types.CaseLaw
    }
    
    // 合同关键词
    if s.containsAny(content, []string{"合同", "协议", "契约"}) {
        return types.Contract
    }
    
    // 法律意见书关键词
    if s.containsAny(content, []string{"法律意见书", "律师函", "法律分析"}) {
        return types.LegalOpinion
    }
    
    // 司法解释关键词
    if s.containsAny(content, []string{"司法解释", "最高人民法院", "最高人民检察院"}) {
        return types.Interpretation
    }
    
    // 默认为法律法规
    return types.Statute
}

func (s *LegalDocumentAnalyzer) identifyLegalDomain(ctx context.Context, content string) string {
    // 使用NLP服务进行领域分类
    domains := map[string][]string{
        "contract_law":    {"合同", "协议", "违约", "履行"},
        "criminal_law":    {"犯罪", "刑罚", "刑事", "犯法"},
        "civil_law":       {"民事", "侵权", "损害赔偿", "人身权"},
        "commercial_law":  {"商业", "公司", "企业", "商务"},
        "labor_law":       {"劳动", "工作", "雇佣", "劳务"},
        "intellectual_property": {"知识产权", "专利", "商标", "著作权"},
    }
    
    maxScore := 0
    bestDomain := "general"
    
    for domain, keywords := range domains {
        score := 0
        for _, keyword := range keywords {
            if strings.Contains(content, keyword) {
                score++
            }
        }
        if score > maxScore {
            maxScore = score
            bestDomain = domain
        }
    }
    
    return bestDomain
}

func (s *LegalDocumentAnalyzer) extractStatuteReferences(content string) []string {
    // 正则表达式匹配法条引用
    patterns := []string{
        `《[^》]+》第\d+条`,
        `第\d+条第\d+款`,
        `[^》]+法第\d+条`,
    }
    
    var references []string
    for _, pattern := range patterns {
        re := regexp.MustCompile(pattern)
        matches := re.FindAllString(content, -1)
        references = append(references, matches...)
    }
    
    // 去重
    uniqueRefs := make(map[string]bool)
    var result []string
    for _, ref := range references {
        if !uniqueRefs[ref] {
            uniqueRefs[ref] = true
            result = append(result, ref)
        }
    }
    
    return result
}

func (s *LegalDocumentAnalyzer) assessRiskLevel(
    ctx context.Context,
    content string,
    docType types.LegalDocumentType,
) string {
    // 基于文档类型和内容评估风险等级
    riskKeywords := map[string]int{
        "违约":   3,
        "赔偿":   2,
        "责任":   2,
        "处罚":   3,
        "禁止":   2,
        "限制":   1,
        "义务":   1,
    }
    
    totalRisk := 0
    for keyword, risk := range riskKeywords {
        if strings.Contains(content, keyword) {
            totalRisk += risk
        }
    }
    
    // 根据文档类型调整风险等级
    switch docType {
    case types.Contract:
        totalRisk += 1 // 合同天然有风险
    case types.CaseLaw:
        totalRisk += 2 // 判例通常涉及争议
    }
    
    if totalRisk >= 5 {
        return "high"
    } else if totalRisk >= 3 {
        return "medium"
    } else {
        return "low"
    }
}

func (s *LegalDocumentAnalyzer) containsAny(text string, keywords []string) bool {
    for _, keyword := range keywords {
        if strings.Contains(text, keyword) {
            return true
        }
    }
    return false
}

func (s *LegalDocumentAnalyzer) extractKeyConcepts(
    ctx context.Context,
    content string,
) ([]string, error) {
    // 使用NLP服务提取关键概念
    request := &types.NLPRequest{
        Text: content,
        Task: "key_concept_extraction",
        Parameters: map[string]interface{}{
            "max_concepts": 10,
            "domain":      "legal",
        },
    }
    
    response, err := s.nlpService.Process(ctx, request)
    if err != nil {
        return nil, err
    }
    
    concepts, ok := response.Result["concepts"].([]string)
    if !ok {
        return []string{}, nil
    }
    
    return concepts, nil
}

func (s *LegalDocumentAnalyzer) checkCompliance(
    ctx context.Context,
    content string,
    docType types.LegalDocumentType,
) []string {
    var flags []string
    
    // 检查是否包含敏感信息
    sensitivePatterns := []string{
        `\d{15}|\d{18}`,           // 身份证号
        `\d{4}-\d{4}-\d{4}-\d{4}`, // 银行卡号格式
        `1[3-9]\d{9}`,             // 手机号
    }
    
    for _, pattern := range sensitivePatterns {
        re := regexp.MustCompile(pattern)
        if re.MatchString(content) {
            flags = append(flags, "contains_personal_info")
            break
        }
    }
    
    // 检查是否需要特殊处理
    if docType == types.Contract {
        flags = append(flags, "requires_contract_review")
    }
    
    if docType == types.CaseLaw {
        flags = append(flags, "requires_case_analysis")
    }
    
    return flags
}

func (s *LegalDocumentAnalyzer) assessSensitivityLevel(
    docType types.LegalDocumentType,
    riskLevel string,
) string {
    // 基于文档类型和风险等级评估敏感度
    baseLevel := map[types.LegalDocumentType]int{
        types.Contract:       3,
        types.CaseLaw:        2,
        types.LegalOpinion:   2,
        types.Statute:        1,
        types.Regulation:     1,
        types.Interpretation: 1,
    }
    
    level := baseLevel[docType]
    
    // 根据风险等级调整
    switch riskLevel {
    case "high":
        level += 2
    case "medium":
        level += 1
    }
    
    if level >= 4 {
        return "high"
    } else if level >= 2 {
        return "medium"
    } else {
        return "low"
    }
}

func (s *LegalDocumentAnalyzer) extractTemporalInfo(
    content string,
    metadata *types.LegalDocumentMetadata,
) {
    // 提取生效日期
    effectiveDatePattern := `(\d{4})年(\d{1,2})月(\d{1,2})日.*?生效`
    re := regexp.MustCompile(effectiveDatePattern)
    if matches := re.FindStringSubmatch(content); len(matches) >= 4 {
        if date, err := time.Parse("2006-1-2", matches[1]+"-"+matches[2]+"-"+matches[3]); err == nil {
            metadata.EffectiveDate = &date
        }
    }
    
    // 提取失效日期
    expiryDatePattern := `(\d{4})年(\d{1,2})月(\d{1,2})日.*?失效`
    re = regexp.MustCompile(expiryDatePattern)
    if matches := re.FindStringSubmatch(content); len(matches) >= 4 {
        if date, err := time.Parse("2006-1-2", matches[1]+"-"+matches[2]+"-"+matches[3]); err == nil {
            metadata.ExpiryDate = &date
        }
    }
}

func (s *LegalDocumentAnalyzer) extractAuthorityInfo(
    content string,
    metadata *types.LegalDocumentMetadata,
) {
    // 识别发布机构
    authorities := map[string]string{
        "全国人民代表大会":     "national_congress",
        "国务院":           "state_council",
        "最高人民法院":       "supreme_court",
        "最高人民检察院":      "supreme_procuratorate",
        "司法部":           "ministry_of_justice",
    }
    
    for authority, code := range authorities {
        if strings.Contains(content, authority) {
            metadata.LegalAuthority = authority
            metadata.AuthorityLevel = code
            break
        }
    }
    
    // 生成引用格式
    if metadata.LegalAuthority != "" {
        metadata.CitationFormat = s.generateCitationFormat(content, metadata.LegalAuthority)
    }
}

func (s *LegalDocumentAnalyzer) generateCitationFormat(content, authority string) string {
    // 简单的引用格式生成逻辑
    // 实际实现中应该根据法律文献引用规范生成
    return fmt.Sprintf("%s发布文件", authority)
}
```

### 4. 外部判例检索服务

```go
// internal/application/service/external_case_search_service.go
package service

import (
    "context"
    "crypto/md5"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/Tencent/aiplusall-kb/internal/types"
    "github.com/Tencent/aiplusall-kb/internal/types/interfaces"
    "github.com/Tencent/aiplusall-kb/internal/logger"
)

type ExternalCaseSearchService struct {
    engines           map[types.ExternalSearchEngineType]types.ExternalSearchEngine
    cacheRepo         interfaces.CacheRepository
    knowledgeService  interfaces.KnowledgeService
    legalAnalyzer     interfaces.LegalDocumentAnalyzer
    webSearchService  interfaces.WebSearchService
    config           *ExternalSearchConfig
}

type ExternalSearchConfig struct {
    EnableCache       bool          `yaml:"enable_cache"`
    CacheTTL         time.Duration `yaml:"cache_ttl"`
    MaxConcurrent    int           `yaml:"max_concurrent"`
    RequestTimeout   time.Duration `yaml:"request_timeout"`
    DefaultEngines   []types.ExternalSearchEngineType `yaml:"default_engines"`
}

func NewExternalCaseSearchService(
    cacheRepo interfaces.CacheRepository,
    knowledgeService interfaces.KnowledgeService,
    legalAnalyzer interfaces.LegalDocumentAnalyzer,
    webSearchService interfaces.WebSearchService,
    config *ExternalSearchConfig,
) *ExternalCaseSearchService {
    return &ExternalCaseSearchService{
        engines:          make(map[types.ExternalSearchEngineType]types.ExternalSearchEngine),
        cacheRepo:        cacheRepo,
        knowledgeService: knowledgeService,
        legalAnalyzer:    legalAnalyzer,
        webSearchService: webSearchService,
        config:          config,
    }
}

func (s *ExternalCaseSearchService) RegisterEngine(
    engineType types.ExternalSearchEngineType,
    engine types.ExternalSearchEngine,
) error {
    if engine == nil {
        return fmt.Errorf("engine cannot be nil")
    }
    
    // 验证引擎配置
    if err := engine.ValidateConfig(); err != nil {
        return fmt.Errorf("engine validation failed: %w", err)
    }
    
    s.engines[engineType] = engine
    logger.Infof(context.Background(), "Registered external search engine: %s", engineType)
    return nil
}

func (s *ExternalCaseSearchService) SearchCases(
    ctx context.Context,
    query *types.CaseSearchQuery,
) (*types.ExternalSearchResult, error) {
    // 1. 检查缓存
    if s.config.EnableCache {
        if cached, err := s.getCachedResult(ctx, query); err == nil {
            logger.Infof(ctx, "External search cache hit for query: %s", query.Query)
            return cached, nil
        }
    }
    
    // 2. 确定使用的搜索引擎
    engines := s.selectEngines(query.EngineType)
    if len(engines) == 0 {
        return nil, fmt.Errorf("no available search engines")
    }
    
    // 3. 并行搜索
    results := make(chan *engineResult, len(engines))
    for _, engineType := range engines {
        go s.searchWithEngine(ctx, engineType, query, results)
    }
    
    // 4. 收集结果
    var allCases []*types.ExternalCaseData
    var usedEngine types.ExternalSearchEngineType
    searchStart := time.Now()
    
    for i := 0; i < len(engines); i++ {
        select {
        case result := <-results:
            if result.err != nil {
                logger.Warnf(ctx, "Engine %s search failed: %v", result.engineType, result.err)
                continue
            }
            allCases = append(allCases, result.cases...)
            if usedEngine == "" {
                usedEngine = result.engineType
            }
        case <-time.After(s.config.RequestTimeout):
            logger.Warnf(ctx, "External search timeout after %v", s.config.RequestTimeout)
            break
        }
    }
    
    // 5. 结果去重和排序
    uniqueCases := s.deduplicateAndRank(allCases, query)
    
    // 6. 构建最终结果
    searchResult := &types.ExternalSearchResult{
        Query:      query,
        Cases:      uniqueCases,
        TotalFound: len(uniqueCases),
        SearchTime: time.Since(searchStart),
        EngineUsed: usedEngine,
        CacheHit:   false,
    }
    
    // 7. 缓存结果
    if s.config.EnableCache {
        s.cacheResult(ctx, query, searchResult)
    }
    
    return searchResult, nil
}

type engineResult struct {
    engineType types.ExternalSearchEngineType
    cases      []*types.ExternalCaseData
    err        error
}

func (s *ExternalCaseSearchService) searchWithEngine(
    ctx context.Context,
    engineType types.ExternalSearchEngineType,
    query *types.CaseSearchQuery,
    results chan<- *engineResult,
) {
    engine, exists := s.engines[engineType]
    if !exists {
        results <- &engineResult{
            engineType: engineType,
            err:        fmt.Errorf("engine %s not found", engineType),
        }
        return
    }
    
    // 执行搜索 - 使用扩展的接口方法
    rawResult, err := engine.SearchCases(ctx, query)
    if err != nil {
        results <- &engineResult{
            engineType: engineType,
            err:        err,
        }
        return
    }
    
    // 解析案例数据
    cases, err := engine.ParseCaseData(ctx, rawResult)
    if err != nil {
        results <- &engineResult{
            engineType: engineType,
            err:        err,
        }
        return
    }
    
    results <- &engineResult{
        engineType: engineType,
        cases:      cases,
        err:        nil,
    }
}

func (s *ExternalCaseSearchService) selectEngines(
    preferredEngine types.ExternalSearchEngineType,
) []types.ExternalSearchEngineType {
    if preferredEngine != "" {
        if _, exists := s.engines[preferredEngine]; exists {
            return []types.ExternalSearchEngineType{preferredEngine}
        }
    }
    
    // 使用默认引擎列表
    var available []types.ExternalSearchEngineType
    for _, engineType := range s.config.DefaultEngines {
        if _, exists := s.engines[engineType]; exists {
            available = append(available, engineType)
        }
    }
    
    return available
}

func (s *ExternalCaseSearchService) deduplicateAndRank(
    cases []*types.ExternalCaseData,
    query *types.CaseSearchQuery,
) []*types.ExternalCaseData {
    // 去重逻辑：基于案件编号
    seen := make(map[string]bool)
    var unique []*types.ExternalCaseData
    
    for _, caseData := range cases {
        if caseData.CaseNumber != "" && seen[caseData.CaseNumber] {
            continue
        }
        seen[caseData.CaseNumber] = true
        unique = append(unique, caseData)
    }
    
    // 排序：按相似度评分降序
    sort.Slice(unique, func(i, j int) bool {
        return unique[i].SimilarityScore > unique[j].SimilarityScore
    })
    
    // 限制结果数量
    if len(unique) > query.MaxResults {
        unique = unique[:query.MaxResults]
    }
    
    return unique
}

func (s *ExternalCaseSearchService) getCachedResult(
    ctx context.Context,
    query *types.CaseSearchQuery,
) (*types.ExternalSearchResult, error) {
    cacheKey := s.generateCacheKey(query)
    
    cached, err := s.cacheRepo.Get(ctx, cacheKey)
    if err != nil {
        return nil, err
    }
    
    var result types.ExternalSearchResult
    if err := json.Unmarshal([]byte(cached), &result); err != nil {
        return nil, err
    }
    
    result.CacheHit = true
    return &result, nil
}

func (s *ExternalCaseSearchService) cacheResult(
    ctx context.Context,
    query *types.CaseSearchQuery,
    result *types.ExternalSearchResult,
) {
    cacheKey := s.generateCacheKey(query)
    
    data, err := json.Marshal(result)
    if err != nil {
        logger.Warnf(ctx, "Failed to marshal search result for cache: %v", err)
        return
    }
    
    err = s.cacheRepo.Set(ctx, cacheKey, string(data), s.config.CacheTTL)
    if err != nil {
        logger.Warnf(ctx, "Failed to cache search result: %v", err)
    }
}

func (s *ExternalCaseSearchService) generateCacheKey(query *types.CaseSearchQuery) string {
    // 生成查询的哈希值作为缓存键
    data, _ := json.Marshal(query)
    hash := md5.Sum(data)
    return fmt.Sprintf("external_search:%x", hash)
}

func (s *ExternalCaseSearchService) ImportCase(
    ctx context.Context,
    caseData *types.ExternalCaseData,
    kbID string,
) (*types.Knowledge, error) {
    // 1. 转换为Knowledge格式
    knowledge := &types.Knowledge{
        ID:              generateKnowledgeID(),
        KnowledgeBaseID: kbID,
        Title:           caseData.CaseTitle,
        Description:     fmt.Sprintf("案件编号：%s，审理法院：%s", caseData.CaseNumber, caseData.Court),
        Source:          caseData.SourceURL,
        Type:            "external_case",
    }
    
    // 2. 构建法律元数据
    legalMetadata := &types.LegalDocumentMetadata{
        DocumentType:    types.CaseLaw,
        LegalDomain:     s.inferLegalDomain(caseData.CauseOfAction),
        Jurisdiction:    "china", // 默认中国司法管辖区
        KeyConcepts:     caseData.LegalIssues,
        CaseNumbers:     []string{caseData.CaseNumber},
        LegalAuthority:  caseData.Court,
        AuthorityLevel:  s.inferAuthorityLevel(caseData.Court),
    }
    
    if !caseData.JudgeDate.IsZero() {
        legalMetadata.EffectiveDate = &caseData.JudgeDate
    }
    
    knowledge.LegalMetadata = legalMetadata
    
    // 3. 构建完整内容
    content := s.buildCaseContent(caseData)
    
    // 4. 创建知识条目
    err := s.knowledgeService.CreateKnowledge(ctx, knowledge, []byte(content))
    if err != nil {
        return nil, fmt.Errorf("failed to create knowledge from external case: %w", err)
    }
    
    logger.Infof(ctx, "Successfully imported external case: %s", caseData.CaseNumber)
    return knowledge, nil
}

func (s *ExternalCaseSearchService) buildCaseContent(caseData *types.ExternalCaseData) string {
    var content strings.Builder
    
    content.WriteString(fmt.Sprintf("# %s\n\n", caseData.CaseTitle))
    content.WriteString(fmt.Sprintf("**案件编号**: %s\n", caseData.CaseNumber))
    content.WriteString(fmt.Sprintf("**审理法院**: %s\n", caseData.Court))
    content.WriteString(fmt.Sprintf("**判决日期**: %s\n", caseData.JudgeDate.Format("2006-01-02")))
    content.WriteString(fmt.Sprintf("**案由**: %s\n\n", caseData.CauseOfAction))
    
    if len(caseData.Plaintiff) > 0 {
        content.WriteString(fmt.Sprintf("**原告**: %s\n", strings.Join(caseData.Plaintiff, "、")))
    }
    if len(caseData.Defendant) > 0 {
        content.WriteString(fmt.Sprintf("**被告**: %s\n", strings.Join(caseData.Defendant, "、")))
    }
    
    content.WriteString("\n## 案件事实\n\n")
    content.WriteString(caseData.CaseFacts)
    
    if len(caseData.LegalIssues) > 0 {
        content.WriteString("\n\n## 争议焦点\n\n")
        for i, issue := range caseData.LegalIssues {
            content.WriteString(fmt.Sprintf("%d. %s\n", i+1, issue))
        }
    }
    
    content.WriteString("\n\n## 法院观点\n\n")
    content.WriteString(caseData.CourtOpinion)
    
    content.WriteString("\n\n## 判决结果\n\n")
    content.WriteString(caseData.JudgmentResult)
    
    if len(caseData.AppliedLaws) > 0 {
        content.WriteString("\n\n## 适用法条\n\n")
        for _, law := range caseData.AppliedLaws {
            content.WriteString(fmt.Sprintf("- %s\n", law))
        }
    }
    
    content.WriteString(fmt.Sprintf("\n\n---\n*来源: [裁判文书网](%s)*", caseData.SourceURL))
    
    return content.String()
}

func (s *ExternalCaseSearchService) inferLegalDomain(causeOfAction string) string {
    domainKeywords := map[string]string{
        "合同":   "contract_law",
        "侵权":   "tort_law",
        "劳动":   "labor_law",
        "婚姻":   "family_law",
        "继承":   "inheritance_law",
        "物权":   "property_law",
        "知识产权": "intellectual_property",
        "公司":   "corporate_law",
        "刑事":   "criminal_law",
    }
    
    for keyword, domain := range domainKeywords {
        if strings.Contains(causeOfAction, keyword) {
            return domain
        }
    }
    
    return "general"
}

func (s *ExternalCaseSearchService) inferAuthorityLevel(court string) string {
    if strings.Contains(court, "最高人民法院") {
        return "supreme_court"
    }
    if strings.Contains(court, "高级人民法院") {
        return "high_court"
    }
    if strings.Contains(court, "中级人民法院") {
        return "intermediate_court"
    }
    return "basic_court"
}

func generateKnowledgeID() string {
    return fmt.Sprintf("know-%d", time.Now().UnixNano())
}
```

```go
// internal/application/service/graph_engine_manager.go
package service

import (
    "context"
    "fmt"
    "sync"
    
    "github.com/Tencent/aiplusall-kb/internal/config"
    "github.com/Tencent/aiplusall-kb/internal/types"
    "github.com/Tencent/aiplusall-kb/internal/types/interfaces"
    "github.com/Tencent/aiplusall-kb/internal/logger"
)

type GraphEngineManager struct {
    engines map[types.GraphEngineType]types.KnowledgeGraphEngine
    config  *config.Config
    mutex   sync.RWMutex
    
    kbRepo       interfaces.KnowledgeBaseRepository
    taskRepo     interfaces.TaskRepository
    configRepo   interfaces.ConfigRepository
}

func NewGraphEngineManager(
    config *config.Config,
    kbRepo interfaces.KnowledgeBaseRepository,
    taskRepo interfaces.TaskRepository,
    configRepo interfaces.ConfigRepository,
) *GraphEngineManager {
    return &GraphEngineManager{
        engines:    make(map[types.GraphEngineType]types.KnowledgeGraphEngine),
        config:     config,
        kbRepo:     kbRepo,
        taskRepo:   taskRepo,
        configRepo: configRepo,
    }
}

func (m *GraphEngineManager) RegisterEngine(
    engineType types.GraphEngineType,
    engine types.KnowledgeGraphEngine,
) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    
    if engine == nil {
        return fmt.Errorf("engine cannot be nil")
    }
    
    m.engines[engineType] = engine
    logger.Infof(context.Background(), "Registered graph engine: %s", engineType)
    return nil
}

func (m *GraphEngineManager) GetEngine(
    engineType types.GraphEngineType,
) (types.KnowledgeGraphEngine, error) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    engine, exists := m.engines[engineType]
    if !exists {
        return nil, fmt.Errorf("engine type %s not registered", engineType)
    }
    return engine, nil
}

func (m *GraphEngineManager) GetAvailableEngines() []types.EngineInfo {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    var engines []types.EngineInfo
    for engineType, engine := range m.engines {
        info := engine.GetEngineInfo()
        info.Type = engineType
        engines = append(engines, *info)
    }
    return engines
}

func (m *GraphEngineManager) SwitchKnowledgeBaseEngine(
    ctx context.Context,
    kbID string,
    targetEngine types.GraphEngineType,
    config map[string]interface{},
) (string, error) {
    // 1. 验证目标引擎是否可用
    engine, err := m.GetEngine(targetEngine)
    if err != nil {
        return "", fmt.Errorf("target engine not available: %w", err)
    }
    
    // 2. 获取知识库信息
    kb, err := m.kbRepo.GetByID(ctx, kbID)
    if err != nil {
        return "", fmt.Errorf("failed to get knowledge base: %w", err)
    }
    
    // 3. 检查是否支持图谱功能
    if kb.OwnerType == "user" {
        return "", fmt.Errorf("user private knowledge bases do not support knowledge graph")
    }
    
    // 4. 创建切换任务
    task := &types.Task{
        ID:          generateTaskID(),
        Type:        "graph_engine_switch",
        Status:      "pending",
        KnowledgeBaseID: kbID,
        Parameters: map[string]interface{}{
            "target_engine": targetEngine,
            "config":       config,
        },
    }
    
    err = m.taskRepo.Create(ctx, task)
    if err != nil {
        return "", fmt.Errorf("failed to create switch task: %w", err)
    }
    
    // 5. 异步执行切换
    go m.executeSwitchTask(context.Background(), task)
    
    return task.ID, nil
}

func (m *GraphEngineManager) executeSwitchTask(ctx context.Context, task *types.Task) {
    // 更新任务状态为进行中
    task.Status = "running"
    m.taskRepo.Update(ctx, task)
    
    defer func() {
        if r := recover(); r != nil {
            logger.Errorf(ctx, "Graph engine switch task panicked: %v", r)
            task.Status = "failed"
            task.ErrorMessage = fmt.Sprintf("Task panicked: %v", r)
            m.taskRepo.Update(ctx, task)
        }
    }()
    
    kbID := task.KnowledgeBaseID
    targetEngine := types.GraphEngineType(task.Parameters["target_engine"].(string))
    config := task.Parameters["config"].(map[string]interface{})
    
    // 1. 获取当前知识库配置
    kb, err := m.kbRepo.GetByID(ctx, kbID)
    if err != nil {
        m.failTask(ctx, task, fmt.Errorf("failed to get knowledge base: %w", err))
        return
    }
    
    // 2. 备份当前配置
    currentConfig := kb.ExtractConfig
    
    // 3. 获取目标引擎
    engine, err := m.GetEngine(targetEngine)
    if err != nil {
        m.failTask(ctx, task, fmt.Errorf("failed to get target engine: %w", err))
        return
    }
    
    // 4. 准备新配置
    newExtractConfig := &types.ExtractConfig{
        Enabled:    currentConfig.Enabled,
        EngineType: targetEngine,
        EngineConfig: config,
    }
    
    // 5. 更新知识库配置
    kb.ExtractConfig = newExtractConfig
    err = m.kbRepo.Update(ctx, kb)
    if err != nil {
        m.failTask(ctx, task, fmt.Errorf("failed to update knowledge base config: %w", err))
        return
    }
    
    // 6. 重建图谱
    err = engine.RebuildGraph(ctx, kbID, 100) // 批量大小100
    if err != nil {
        // 回滚配置
        kb.ExtractConfig = currentConfig
        m.kbRepo.Update(ctx, kb)
        m.failTask(ctx, task, fmt.Errorf("failed to rebuild graph: %w", err))
        return
    }
    
    // 7. 完成任务
    task.Status = "completed"
    task.CompletedAt = timeNow()
    m.taskRepo.Update(ctx, task)
    
    logger.Infof(ctx, "Successfully switched knowledge base %s to engine %s", kbID, targetEngine)
}

func (m *GraphEngineManager) failTask(ctx context.Context, task *types.Task, err error) {
    task.Status = "failed"
    task.ErrorMessage = err.Error()
    task.CompletedAt = timeNow()
    m.taskRepo.Update(ctx, task)
    logger.Errorf(ctx, "Graph engine switch task failed: %v", err)
}

func (m *GraphEngineManager) RebuildGraphAsync(
    ctx context.Context,
    kbID string,
    engineType types.GraphEngineType,
    batchSize int,
    config map[string]interface{},
) (string, error) {
    // 创建重建任务
    task := &types.Task{
        ID:              generateTaskID(),
        Type:            "graph_rebuild",
        Status:          "pending",
        KnowledgeBaseID: kbID,
        Parameters: map[string]interface{}{
            "engine_type": engineType,
            "batch_size":  batchSize,
            "config":      config,
        },
    }
    
    err := m.taskRepo.Create(ctx, task)
    if err != nil {
        return "", fmt.Errorf("failed to create rebuild task: %w", err)
    }
    
    // 异步执行重建
    go m.executeRebuildTask(context.Background(), task)
    
    return task.ID, nil
}

func (m *GraphEngineManager) executeRebuildTask(ctx context.Context, task *types.Task) {
    // 实现图谱重建逻辑
    // 类似于 executeSwitchTask，但专注于重建而不是切换
    // 这里省略具体实现...
}

func generateTaskID() string {
    // 生成唯一任务ID的实现
    return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

func timeNow() *time.Time {
    now := time.Now()
    return &now
}
```

### 5. 外部搜索引擎适配器

#### Perplexity.ai 法律搜索引擎适配器

```go
// internal/application/service/web_search/perplexity_legal.go
package web_search

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "regexp"
    "strings"
    "time"
    
    "github.com/Tencent/aiplusall-kb/internal/config"
    "github.com/Tencent/aiplusall-kb/internal/types"
    "github.com/Tencent/aiplusall-kb/internal/types/interfaces"
    "github.com/Tencent/aiplusall-kb/internal/logger"
)

// PerplexityLegalProvider 扩展现有的 PerplexityProvider 以支持法律案例搜索
type PerplexityLegalProvider struct {
    *PerplexityProvider // 继承现有的 Perplexity 提供者
    legalConfig *PerplexityLegalConfig
}

type PerplexityLegalConfig struct {
    LegalModel       string        `yaml:"legal_model"`
    LegalPromptTemplate string     `yaml:"legal_prompt_template"`
    MaxCaseResults   int           `yaml:"max_case_results"`
    CaseParsingRules []string      `yaml:"case_parsing_rules"`
}

func NewPerplexityLegalProvider(cfg config.WebSearchProviderConfig) (interfaces.WebSearchProvider, error) {
    // 首先创建基础的 Perplexity 提供者
    baseProvider, err := NewPerplexityProvider(cfg)
    if err != nil {
        return nil, fmt.Errorf("failed to create base perplexity provider: %w", err)
    }
    
    legalConfig := &PerplexityLegalConfig{
        LegalModel:       "sonar-pro", // 使用更强的模型进行法律搜索
        MaxCaseResults:   10,
        LegalPromptTemplate: `你是一个专业的中国法律案例搜索助手。请根据以下查询条件，搜索中国裁判文书网等权威法律数据库中的相关判例：

查询条件：
%s

请返回最相关的判例，每个判例包含：
1. 案件编号
2. 案件标题  
3. 审理法院
4. 判决日期
5. 案由
6. 当事人（原告、被告）
7. 案件事实摘要
8. 争议焦点
9. 法院观点
10. 判决结果
11. 适用法条
12. 来源链接

请确保信息准确，优先返回权威性高、相关性强的判例。`,
    }
    
    return &PerplexityLegalProvider{
        PerplexityProvider: baseProvider.(*PerplexityProvider),
        legalConfig:       legalConfig,
    }, nil
}

// SearchCases 实现法律案例专门搜索
func (p *PerplexityLegalProvider) SearchCases(
    ctx context.Context,
    query *types.CaseSearchQuery,
) (*types.RawSearchResult, error) {
    // 构建法律专业搜索提示词
    prompt := p.buildLegalSearchPrompt(query)
    
    // 使用基础 Perplexity 功能进行搜索，但使用法律专业提示词
    results, err := p.Search(ctx, prompt, query.MaxResults, true)
    if err != nil {
        return nil, fmt.Errorf("perplexity legal search failed: %w", err)
    }
    
    // 将 WebSearchResult 转换为 RawSearchResult
    content := p.combineSearchResults(results)
    
    return &types.RawSearchResult{
        Content:    content,
        Source:     "perplexity_legal",
        Timestamp:  time.Now(),
        Metadata: map[string]interface{}{
            "query_type":    "legal_case_search",
            "results_count": len(results),
            "model_used":    p.legalConfig.LegalModel,
        },
    }, nil
}

func (p *PerplexityLegalProvider) buildLegalSearchPrompt(query *types.CaseSearchQuery) string {
    var queryParts []string
    
    queryParts = append(queryParts, fmt.Sprintf("关键词：%s", query.Query))
    
    if query.CaseType != "" {
        queryParts = append(queryParts, fmt.Sprintf("案件类型：%s", query.CaseType))
    }
    
    if query.Court != "" {
        queryParts = append(queryParts, fmt.Sprintf("审理法院：%s", query.Court))
    }
    
    if query.Jurisdiction != "" {
        queryParts = append(queryParts, fmt.Sprintf("司法管辖区：%s", query.Jurisdiction))
    }
    
    if query.CauseOfAction != "" {
        queryParts = append(queryParts, fmt.Sprintf("案由：%s", query.CauseOfAction))
    }
    
    if len(query.LegalIssues) > 0 {
        queryParts = append(queryParts, fmt.Sprintf("争议焦点：%s", strings.Join(query.LegalIssues, "、")))
    }
    
    if query.DateRange != nil {
        queryParts = append(queryParts, fmt.Sprintf("时间范围：%s 至 %s", 
            query.DateRange.Start.Format("2006-01-02"),
            query.DateRange.End.Format("2006-01-02")))
    }
    
    queryConditions := strings.Join(queryParts, "\n")
    return fmt.Sprintf(p.legalConfig.LegalPromptTemplate, queryConditions)
}

func (p *PerplexityLegalProvider) combineSearchResults(results []*types.WebSearchResult) string {
    var content strings.Builder
    
    for i, result := range results {
        content.WriteString(fmt.Sprintf("=== 搜索结果 %d ===\n", i+1))
        content.WriteString(fmt.Sprintf("标题：%s\n", result.Title))
        content.WriteString(fmt.Sprintf("链接：%s\n", result.URL))
        content.WriteString(fmt.Sprintf("摘要：%s\n", result.Snippet))
        if result.Content != "" {
            content.WriteString(fmt.Sprintf("内容：%s\n", result.Content))
        }
        content.WriteString("\n")
    }
    
    return content.String()
}

// ParseCaseData 解析搜索结果为结构化案例数据
func (p *PerplexityLegalProvider) ParseCaseData(
    ctx context.Context,
    rawResult *types.RawSearchResult,
) ([]*types.ExternalCaseData, error) {
    content := rawResult.Content
    
    // 使用正则表达式和文本解析提取案例信息
    cases := p.extractCasesFromText(content)
    
    // 计算相似度评分
    for _, caseData := range cases {
        caseData.SimilarityScore = p.calculateSimilarityScore(caseData, content)
        caseData.Confidence = 0.85 // Perplexity 的基础置信度
        caseData.ExtractedAt = time.Now()
    }
    
    return cases, nil
}

func (p *PerplexityLegalProvider) extractCasesFromText(content string) []*types.ExternalCaseData {
    var cases []*types.ExternalCaseData
    
    // 分割案例（假设每个案例以数字开头或特定标记分隔）
    casePattern := regexp.MustCompile(`(?m)^(?:=== 搜索结果 \d+ ===|案例 \d+[：:]|判例 \d+[：:]|\d+\.\s*)(.+?)(?=^(?:=== 搜索结果 \d+ ===|案例 \d+[：:]|判例 \d+[：:]|\d+\.\s*)|$)`)
    matches := casePattern.FindAllStringSubmatch(content, -1)
    
    for _, match := range matches {
        if len(match) < 2 {
            continue
        }
        
        caseText := match[1]
        caseData := p.parseSingleCase(caseText)
        if caseData != nil {
            cases = append(cases, caseData)
        }
    }
    
    return cases
}

func (p *PerplexityLegalProvider) parseSingleCase(caseText string) *types.ExternalCaseData {
    caseData := &types.ExternalCaseData{}
    
    // 提取案件编号
    if caseNumber := p.extractField(caseText, `案件编号[：:]\s*([^\n]+)`); caseNumber != "" {
        caseData.CaseNumber = strings.TrimSpace(caseNumber)
    }
    
    // 提取案件标题
    if caseTitle := p.extractField(caseText, `(?:案件标题|标题)[：:]\s*([^\n]+)`); caseTitle != "" {
        caseData.CaseTitle = strings.TrimSpace(caseTitle)
    }
    
    // 提取审理法院
    if court := p.extractField(caseText, `审理法院[：:]\s*([^\n]+)`); court != "" {
        caseData.Court = strings.TrimSpace(court)
    }
    
    // 提取判决日期
    if dateStr := p.extractField(caseText, `判决日期[：:]\s*([^\n]+)`); dateStr != "" {
        if date, err := time.Parse("2006-01-02", strings.TrimSpace(dateStr)); err == nil {
            caseData.JudgeDate = date
        }
    }
    
    // 提取案由
    if causeOfAction := p.extractField(caseText, `案由[：:]\s*([^\n]+)`); causeOfAction != "" {
        caseData.CauseOfAction = strings.TrimSpace(causeOfAction)
    }
    
    // 提取来源链接
    if sourceURL := p.extractField(caseText, `(?:来源链接|链接)[：:]\s*([^\n\s]+)`); sourceURL != "" {
        caseData.SourceURL = strings.TrimSpace(sourceURL)
    }
    
    // 提取案件事实
    if caseFacts := p.extractField(caseText, `案件事实[：:]?\s*([^法院观点]{1,500})`); caseFacts != "" {
        caseData.CaseFacts = strings.TrimSpace(caseFacts)
    }
    
    // 提取法院观点
    if courtOpinion := p.extractField(caseText, `法院观点[：:]?\s*([^判决结果]{1,300})`); courtOpinion != "" {
        caseData.CourtOpinion = strings.TrimSpace(courtOpinion)
    }
    
    // 提取判决结果
    if judgmentResult := p.extractField(caseText, `判决结果[：:]?\s*([^适用法条]{1,200})`); judgmentResult != "" {
        caseData.JudgmentResult = strings.TrimSpace(judgmentResult)
    }
    
    // 基本验证
    if caseData.CaseNumber == "" && caseData.CaseTitle == "" {
        return nil
    }
    
    return caseData
}

func (p *PerplexityLegalProvider) extractField(text, pattern string) string {
    re := regexp.MustCompile(pattern)
    matches := re.FindStringSubmatch(text)
    if len(matches) > 1 {
        return matches[1]
    }
    return ""
}

func (p *PerplexityLegalProvider) calculateSimilarityScore(caseData *types.ExternalCaseData, originalQuery string) float64 {
    // 简单的相似度计算，实际实现中应该使用更复杂的算法
    score := 0.0
    
    // 基于关键词匹配计算相似度
    queryLower := strings.ToLower(originalQuery)
    
    if strings.Contains(queryLower, strings.ToLower(caseData.CauseOfAction)) {
        score += 0.3
    }
    
    if strings.Contains(queryLower, strings.ToLower(caseData.Court)) {
        score += 0.2
    }
    
    // 基于案件事实的相似度
    if len(caseData.CaseFacts) > 0 {
        commonWords := p.countCommonWords(queryLower, strings.ToLower(caseData.CaseFacts))
        score += float64(commonWords) * 0.1
    }
    
    // 确保分数在0-1之间
    if score > 1.0 {
        score = 1.0
    }
    
    return score
}

func (p *PerplexityLegalProvider) countCommonWords(text1, text2 string) int {
    words1 := strings.Fields(text1)
    words2 := strings.Fields(text2)
    
    wordSet := make(map[string]bool)
    for _, word := range words1 {
        if len(word) > 1 { // 忽略单字符
            wordSet[word] = true
        }
    }
    
    commonCount := 0
    for _, word := range words2 {
        if len(word) > 1 && wordSet[word] {
            commonCount++
        }
    }
    
    return commonCount
}

func (p *PerplexityLegalProvider) GetEngineInfo() *types.ExternalEngineInfo {
    return &types.ExternalEngineInfo{
        Name:        "Perplexity.ai Legal",
        Description: "基于 Perplexity.ai 的智能法律案例搜索引擎",
        Version:     "1.0.0",
        Features: []string{
            "实时网络搜索",
            "AI 智能理解",
            "多源数据整合",
            "自然语言查询",
            "法律专业优化",
        },
        SupportedSources: []string{
            "裁判文书网",
            "法律数据库",
            "学术文献",
            "新闻报道",
        },
        Limitations: []string{
            "依赖网络连接",
            "API 调用限制",
            "结果准确性依赖 AI 模型",
        },
    }
}

func (p *PerplexityLegalProvider) ValidateConfig() error {
    // 使用基础 Perplexity 提供者的验证逻辑
    if p.PerplexityProvider == nil {
        return fmt.Errorf("base perplexity provider is not initialized")
    }
    
    if p.legalConfig.LegalModel == "" {
        return fmt.Errorf("legal model is required")
    }
    
    if p.legalConfig.LegalPromptTemplate == "" {
        return fmt.Errorf("legal prompt template is required")
    }
    
    return nil
}
```

#### Browser-use 引擎适配器

```go
// internal/application/service/web_search/browser_use.go
package web_search

import (
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
    "regexp"
    "strings"
    "time"
    
    "github.com/Tencent/aiplusall-kb/internal/config"
    "github.com/Tencent/aiplusall-kb/internal/types"
    "github.com/Tencent/aiplusall-kb/internal/types/interfaces"
    "github.com/Tencent/aiplusall-kb/internal/logger"
)

// BrowserUseProvider 实现基于浏览器自动化的法律案例搜索
type BrowserUseProvider struct {
    config     *BrowserUseConfig
    pythonPath string
    scriptPath string
}

type BrowserUseConfig struct {
    PythonPath     string        `yaml:"python_path"`
    ScriptPath     string        `yaml:"script_path"`
    Timeout        time.Duration `yaml:"timeout"`
    MaxRetries     int           `yaml:"max_retries"`
    HeadlessMode   bool          `yaml:"headless_mode"`
    UserAgent      string        `yaml:"user_agent"`
    RequestDelay   time.Duration `yaml:"request_delay"`
}

type BrowserUseRequest struct {
    Query          string            `json:"query"`
    TargetSites    []string          `json:"target_sites"`
    MaxResults     int               `json:"max_results"`
    SearchTimeout  int               `json:"search_timeout"`
    HeadlessMode   bool              `json:"headless_mode"`
    UserAgent      string            `json:"user_agent"`
    RequestDelay   int               `json:"request_delay"`
    ExtraParams    map[string]string `json:"extra_params"`
}

type BrowserUseResponse struct {
    Success    bool                    `json:"success"`
    Results    []BrowserUseResult      `json:"results"`
    Error      string                  `json:"error,omitempty"`
    Metadata   BrowserUseMetadata      `json:"metadata"`
}

type BrowserUseResult struct {
    Title       string            `json:"title"`
    URL         string            `json:"url"`
    Content     string            `json:"content"`
    Metadata    map[string]string `json:"metadata"`
    Timestamp   string            `json:"timestamp"`
    Confidence  float64           `json:"confidence"`
}

type BrowserUseMetadata struct {
    SearchTime    float64 `json:"search_time"`
    PagesVisited  int     `json:"pages_visited"`
    ResultsFound  int     `json:"results_found"`
    ErrorCount    int     `json:"error_count"`
}

func NewBrowserUseProvider(cfg config.WebSearchProviderConfig) (interfaces.WebSearchProvider, error) {
    config := &BrowserUseConfig{
        PythonPath:   cfg.GetString("python_path", "python3"),
        ScriptPath:   cfg.GetString("script_path", "./scripts/browser_use_search.py"),
        Timeout:      cfg.GetDuration("timeout", 60*time.Second),
        MaxRetries:   cfg.GetInt("max_retries", 3),
        HeadlessMode: cfg.GetBool("headless_mode", true),
        UserAgent:    cfg.GetString("user_agent", "Mozilla/5.0 (compatible; aiplusall-kb/1.0)"),
        RequestDelay: cfg.GetDuration("request_delay", 2*time.Second),
    }
    
    return &BrowserUseProvider{
        config:     config,
        pythonPath: config.PythonPath,
        scriptPath: config.ScriptPath,
    }, nil
}

// Name 返回提供者名称
func (p *BrowserUseProvider) Name() string {
    return "browser_use"
}

// Search 实现 WebSearchProvider 接口的基础搜索功能
func (p *BrowserUseProvider) Search(
    ctx context.Context,
    query string,
    maxResults int,
    includeDate bool,
) ([]*types.WebSearchResult, error) {
    // 构建基础搜索请求
    request := &BrowserUseRequest{
        Query: query,
        TargetSites: []string{
            "wenshu.court.gov.cn",  // 裁判文书网
            "pkulaw.com",           // 北大法宝
            "lawinfochina.com",     // 法律信息网
        },
        MaxResults:    maxResults,
        SearchTimeout: int(p.config.Timeout.Seconds()),
        HeadlessMode:  p.config.HeadlessMode,
        UserAgent:     p.config.UserAgent,
        RequestDelay:  int(p.config.RequestDelay.Milliseconds()),
    }
    
    // 调用 Python 脚本
    response, err := p.callBrowserUseScript(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("browser-use script failed: %w", err)
    }
    
    if !response.Success {
        return nil, fmt.Errorf("browser-use search failed: %s", response.Error)
    }
    
    // 转换为 WebSearchResult
    var results []*types.WebSearchResult
    for _, result := range response.Results {
        webResult := &types.WebSearchResult{
            Title:   result.Title,
            URL:     result.URL,
            Snippet: p.extractSnippet(result.Content),
            Content: result.Content,
            Source:  "browser_use",
        }
        
        if timestamp, err := time.Parse(time.RFC3339, result.Timestamp); err == nil {
            webResult.PublishedAt = &timestamp
        }
        
        results = append(results, webResult)
    }
    
    return results, nil
}

// SearchCases 实现法律案例专门搜索
func (p *BrowserUseProvider) SearchCases(
    ctx context.Context,
    query *types.CaseSearchQuery,
) (*types.RawSearchResult, error) {
    // 构建搜索请求
    request := &BrowserUseRequest{
        Query: p.buildSearchQuery(query),
        TargetSites: []string{
            "wenshu.court.gov.cn",  // 裁判文书网
            "pkulaw.com",           // 北大法宝
            "lawinfochina.com",     // 法律信息网
        },
        MaxResults:    query.MaxResults,
        SearchTimeout: int(p.config.Timeout.Seconds()),
        HeadlessMode:  p.config.HeadlessMode,
        UserAgent:     p.config.UserAgent,
        RequestDelay:  int(p.config.RequestDelay.Milliseconds()),
        ExtraParams: map[string]string{
            "case_type":      query.CaseType,
            "court":          query.Court,
            "jurisdiction":   query.Jurisdiction,
            "cause_of_action": query.CauseOfAction,
        },
    }
    
    // 调用 Python 脚本
    response, err := p.callBrowserUseScript(ctx, request)
    if err != nil {
        return nil, fmt.Errorf("browser-use script failed: %w", err)
    }
    
    if !response.Success {
        return nil, fmt.Errorf("browser-use search failed: %s", response.Error)
    }
    
    // 转换结果
    content, err := json.Marshal(response.Results)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal results: %w", err)
    }
    
    return &types.RawSearchResult{
        Content:   string(content),
        Source:    "browser_use",
        Timestamp: time.Now(),
        Metadata: map[string]interface{}{
            "search_time":    response.Metadata.SearchTime,
            "pages_visited":  response.Metadata.PagesVisited,
            "results_found":  response.Metadata.ResultsFound,
            "error_count":    response.Metadata.ErrorCount,
        },
    }, nil
}

func (p *BrowserUseProvider) buildSearchQuery(query *types.CaseSearchQuery) string {
    var queryParts []string
    
    // 基础查询
    queryParts = append(queryParts, query.Query)
    
    // 添加案件类型
    if query.CaseType != "" {
        queryParts = append(queryParts, query.CaseType)
    }
    
    // 添加案由
    if query.CauseOfAction != "" {
        queryParts = append(queryParts, query.CauseOfAction)
    }
    
    // 添加法院
    if query.Court != "" {
        queryParts = append(queryParts, query.Court)
    }
    
    // 添加争议焦点
    if len(query.LegalIssues) > 0 {
        queryParts = append(queryParts, strings.Join(query.LegalIssues, " "))
    }
    
    return strings.Join(queryParts, " ")
}

func (p *BrowserUseProvider) extractSnippet(content string) string {
    // 提取前200个字符作为摘要
    if len(content) > 200 {
        return content[:200] + "..."
    }
    return content
}

func (p *BrowserUseProvider) callBrowserUseScript(
    ctx context.Context,
    request *BrowserUseRequest,
) (*BrowserUseResponse, error) {
    // 序列化请求
    requestJSON, err := json.Marshal(request)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal request: %w", err)
    }
    
    // 准备命令
    cmd := exec.CommandContext(ctx, p.pythonPath, p.scriptPath)
    cmd.Stdin = strings.NewReader(string(requestJSON))
    
    // 执行命令
    output, err := cmd.Output()
    if err != nil {
        return nil, fmt.Errorf("script execution failed: %w", err)
    }
    
    // 解析响应
    var response BrowserUseResponse
    if err := json.Unmarshal(output, &response); err != nil {
        return nil, fmt.Errorf("failed to parse response: %w", err)
    }
    
    return &response, nil
}

// ParseCaseData 解析搜索结果为结构化案例数据
func (p *BrowserUseProvider) ParseCaseData(
    ctx context.Context,
    rawResult *types.RawSearchResult,
) ([]*types.ExternalCaseData, error) {
    var browserResults []BrowserUseResult
    if err := json.Unmarshal([]byte(rawResult.Content), &browserResults); err != nil {
        return nil, fmt.Errorf("failed to unmarshal browser results: %w", err)
    }
    
    var cases []*types.ExternalCaseData
    for _, result := range browserResults {
        caseData := p.parseBrowserResult(result)
        if caseData != nil {
            caseData.SimilarityScore = result.Confidence
            caseData.ExtractedAt = time.Now()
            cases = append(cases, caseData)
        }
    }
    
    return cases, nil
}

func (p *BrowserUseProvider) parseBrowserResult(result BrowserUseResult) *types.ExternalCaseData {
    caseData := &types.ExternalCaseData{
        CaseTitle:  result.Title,
        SourceURL:  result.URL,
        Confidence: result.Confidence,
    }
    
    content := result.Content
    
    // 提取案件编号
    if caseNumber := p.extractWithRegex(content, `案件编号[：:]\s*([^\n\s]+)`); caseNumber != "" {
        caseData.CaseNumber = caseNumber
    }
    
    // 提取审理法院
    if court := p.extractWithRegex(content, `审理法院[：:]\s*([^\n]+)`); court != "" {
        caseData.Court = strings.TrimSpace(court)
    }
    
    // 提取判决日期
    if dateStr := p.extractWithRegex(content, `判决日期[：:]\s*(\d{4}-\d{2}-\d{2})`); dateStr != "" {
        if date, err := time.Parse("2006-01-02", dateStr); err == nil {
            caseData.JudgeDate = date
        }
    }
    
    // 提取案由
    if causeOfAction := p.extractWithRegex(content, `案由[：:]\s*([^\n]+)`); causeOfAction != "" {
        caseData.CauseOfAction = strings.TrimSpace(causeOfAction)
    }
    
    // 提取当事人信息
    if plaintiff := p.extractWithRegex(content, `原告[：:]\s*([^\n]+)`); plaintiff != "" {
        caseData.Plaintiff = strings.Split(strings.TrimSpace(plaintiff), "、")
    }
    
    if defendant := p.extractWithRegex(content, `被告[：:]\s*([^\n]+)`); defendant != "" {
        caseData.Defendant = strings.Split(strings.TrimSpace(defendant), "、")
    }
    
    // 提取案件事实（取前500字符）
    if caseFacts := p.extractWithRegex(content, `案件事实[：:]?\s*([^。]{1,500})`); caseFacts != "" {
        caseData.CaseFacts = strings.TrimSpace(caseFacts)
    }
    
    // 提取法院观点（取前300字符）
    if courtOpinion := p.extractWithRegex(content, `法院[认为观点][：:]?\s*([^。]{1,300})`); courtOpinion != "" {
        caseData.CourtOpinion = strings.TrimSpace(courtOpinion)
    }
    
    // 提取判决结果（取前200字符）
    if judgmentResult := p.extractWithRegex(content, `判决[结果如下][：:]?\s*([^。]{1,200})`); judgmentResult != "" {
        caseData.JudgmentResult = strings.TrimSpace(judgmentResult)
    }
    
    // 从元数据中提取额外信息
    if caseType, exists := result.Metadata["case_type"]; exists {
        caseData.CaseType = caseType
    }
    
    // 基本验证
    if caseData.CaseNumber == "" && caseData.CaseTitle == "" {
        return nil
    }
    
    return caseData
}

func (p *BrowserUseProvider) extractWithRegex(text, pattern string) string {
    re := regexp.MustCompile(pattern)
    matches := re.FindStringSubmatch(text)
    if len(matches) > 1 {
        return matches[1]
    }
    return ""
}

func (p *BrowserUseProvider) GetEngineInfo() *types.ExternalEngineInfo {
    return &types.ExternalEngineInfo{
        Name:        "Browser-use",
        Description: "基于浏览器自动化的法律案例搜索引擎",
        Version:     "1.0.0",
        Features: []string{
            "真实浏览器模拟",
            "JavaScript 支持",
            "反爬虫绕过",
            "多站点搜索",
            "结构化数据提取",
        },
        SupportedSources: []string{
            "裁判文书网",
            "北大法宝",
            "法律信息网",
            "各级法院官网",
        },
        Limitations: []string{
            "搜索速度较慢",
            "资源消耗较大",
            "可能被反爬虫限制",
            "需要 Python 环境",
        },
    }
}

func (p *BrowserUseProvider) ValidateConfig() error {
    if p.config.PythonPath == "" {
        return fmt.Errorf("python path is required")
    }
    
    if p.config.ScriptPath == "" {
        return fmt.Errorf("script path is required")
    }
    
    // 检查 Python 脚本是否存在
    if _, err := exec.LookPath(p.config.PythonPath); err != nil {
        return fmt.Errorf("python executable not found: %w", err)
    }
    
    return nil
}
```

## 正确性属性

*属性是系统应该满足的特征或行为，它们作为人类可读规范和机器可验证正确性保证之间的桥梁。*

### 属性 1: 权限隔离不变性
*对于任何*用户和知识库，用户只能访问其有权限的知识库，且权限级别严格按照所有权>分享>公开的优先级计算
**验证需求: 需求 1.6, 1.7**

### 属性 2: 资源租户一致性
*对于任何*跨租户访问的知识库，系统必须使用资源租户的配置进行检索、存储和计算操作
**验证需求: 需求 4.2, 4.3, 4.4**

### 属性 3: 法律元数据完整性
*对于任何*上传的法律文档，系统必须提取并存储完整的法律元数据，包括文档类型、法律领域、司法管辖区等信息
**验证需求: 需求 2.1, 2.3**

### 属性 4: 分享权限时效性
*对于任何*设置了过期时间的分享，系统必须在过期后自动禁用该分享权限
**验证需求: 需求 1.5**

### 属性 5: 图谱引擎隔离性
*对于任何*用户私有知识库，系统必须禁用知识图谱功能，仅提供向量和关键词检索
**验证需求: 需求 3.6**

### 属性 6: 审计日志完整性
*对于任何*敏感操作，系统必须记录完整的审计日志，包括用户、操作、时间、资源等信息
**验证需求: 需求 2.7, 需求 7.3**

### 属性 7: 数据迁移幂等性
*对于任何*数据库迁移操作，重复执行应该产生相同的结果，不会破坏数据完整性
**验证需求: 需求 6.1, 6.2**

### 属性 8: 缓存一致性
*对于任何*权限变更操作，系统必须及时失效相关缓存，确保权限检查的实时性
**验证需求: 需求 7.4**

### 属性 9: 外部搜索结果一致性
*对于任何*外部判例搜索查询，系统必须返回去重后的结果，且相似度评分准确反映与查询的相关性
**验证需求: 需求 8.3, 8.4, 8.5**

## 错误处理

### 权限错误处理
- 无权限访问返回 403 Forbidden
- 权限检查失败返回 500 Internal Server Error
- 分享过期自动返回 403 Forbidden

### 法律分析错误处理
- 文档类型识别失败使用默认类型
- 元数据提取失败记录警告日志但不阻断流程
- 合规检查失败标记为需要人工审核

### 图谱引擎错误处理
- 引擎切换失败自动回滚配置
- 图谱重建失败保留原有图谱
- 引擎不可用时降级到基础检索

### 数据一致性错误处理
- 跨租户操作失败时回滚所有相关操作
- 缓存失效失败时记录日志但不影响主流程
- 审计日志写入失败时重试3次

## 配置规范

### 系统配置文件结构

基于法律领域专业化需求，系统配置文件应包含以下核心模块：

```yaml
# config/config.yaml - aiplusall-kb 知识库改造完整配置

# 服务器配置
server:
  port: 8080
  host: "0.0.0.0"

# 多层权限模型配置
permission:
  # 权限缓存配置
  cache:
    enabled: true
    ttl: 300s  # 5分钟缓存
    redis_key_prefix: "aiplusall-kb:permission:"
  
  # 分享权限配置
  sharing:
    max_shares_per_kb: 100
    default_expiry_days: 30
    allow_public_sharing: false
    require_approval: true
  
  # 跨租户访问配置
  cross_tenant:
    enabled: true
    audit_all_access: true
    max_concurrent_access: 1000

# 知识库配置 - 针对法律文档优化
knowledge_base:
  # 分块配置 - 法律文档专用
  chunk_size: 700              # 适合法律条款和判决段落
  chunk_overlap: 100           # 确保跨chunk实体关系完整性
  paragraph_aware: true        # 优先保证段落完整性
  language: "zh"               # 中文法律文档
  
  # 法律文档分隔符 - 按优先级排序
  split_markers: 
    - "\n\n\n"      # 大章节分隔
    - "\n\n"        # 段落分隔
    - "。\n"        # 句末换行（法条结尾）
    - "；\n"        # 分号换行（法条款项分隔）
    - "\n第"        # 法条章节标记
    - "\n（"        # 法条项标记
    - "。"          # 中文句号
    - "；"          # 中文分号
    - "："          # 冒号（法条定义）
    - "\n"          # 单换行
    - "，"          # 逗号（兜底）
  
  # 句尾标点识别
  sentence_end_punctuation:
    - "。" - "！" - "？" - "；"
    - "." - "!" - "?" - ";"
  
  # VLM多模态配置
  vlm:
    enabled: true
    model_id: ""  # 使用系统默认
  
  # 默认模型配置
  default_embedding_model_id: "text-embedding-3-small"
  default_summary_model_id: "gpt-4o-mini"
  
  # 问题生成配置
  question_generation:
    enabled: false
    question_count: 3

# 多引擎知识图谱配置
graph_engines:
  # 默认引擎
  default_engine: "aiplusall-kb"
  
  # 引擎配置
  engines:
    aiplusall-kb:
      enabled: true
      description: "aiplusall-kb原生知识图谱引擎"
      neo4j:
        uri: "bolt://localhost:7687"
        username: "neo4j"
        password: "${NEO4J_PASSWORD}"
      features:
        - "Neo4j存储"
        - "实体关系抽取"
        - "异步重建"
    
    graphrag:
      enabled: true
      description: "Microsoft GraphRAG知识图谱引擎"
      config:
        api_key: "${OPENAI_API_KEY}"
        model_name: "gpt-4o-mini"
        community_detection: true
        max_gleanings: 1
      features:
        - "社区检测"
        - "全局查询"
        - "层次化分析"
    
    lightrag:
      enabled: true
      description: "HKUDS LightRAG知识图谱引擎"
      config:
        storage_backend: "neo4j"
        embedding_model: "text-embedding-3-small"
        llm_model: "gpt-4o-mini"
      features:
        - "双层检索"
        - "多存储后端"
        - "轻量化部署"
  
  # 用户私有知识库限制
  user_private_restrictions:
    disable_graph: true
    allowed_search_types: ["vector", "keyword"]

# 法律专业化配置
legal:
  # 法律文档分析配置
  document_analyzer:
    enabled: true
    confidence_threshold: 0.8
    
    # 文档类型识别
    document_types:
      - "statute"        # 法条
      - "case_law"       # 判例
      - "contract"       # 合同
      - "legal_opinion"  # 法律意见书
      - "regulation"     # 法规
      - "interpretation" # 司法解释
    
    # 法律领域分类
    legal_domains:
      - "contract_law"    # 合同法
      - "criminal_law"    # 刑法
      - "civil_law"       # 民法
      - "commercial_law"  # 商法
      - "labor_law"       # 劳动法
      - "intellectual_property" # 知识产权法
    
    # 风险评估配置
    risk_assessment:
      enabled: true
      risk_keywords:
        high: ["违约", "赔偿", "责任", "处罚"]
        medium: ["禁止", "限制", "义务"]
        low: ["建议", "参考"]
  
  # 合规审计配置
  compliance_audit:
    enabled: true
    log_all_access: true
    retention_days: 2555  # 7年
    sensitive_operations:
      - "knowledge_access"
      - "document_download"
      - "sharing_create"
      - "permission_change"

# 外部判例检索配置
external_case_search:
  enabled: true
  
  # 缓存配置
  cache:
    enabled: true
    ttl: 3600s  # 1小时
    max_entries: 10000
  
  # 搜索引擎配置
  engines:
    perplexity_legal:
      enabled: true
      api_key: "${PERPLEXITY_API_KEY}"
      model: "sonar-pro"
      max_tokens: 2048
      temperature: 0.2
      timeout: 30s
      
    browser_use:
      enabled: true
      python_path: "python3"
      script_path: "./scripts/browser_use_search.py"
      headless_mode: true
      timeout: 60s
      request_delay: 2s
      target_sites:
        - "wenshu.court.gov.cn"  # 裁判文书网
        - "pkulaw.com"           # 北大法宝
        - "lawinfochina.com"     # 法律信息网
  
  # 搜索配置
  search:
    max_results: 10
    max_concurrent: 3
    request_timeout: 30s
    default_engines: ["perplexity_legal", "browser_use"]
    
    # 结果处理
    deduplication:
      enabled: true
      similarity_threshold: 0.9
    
    ranking:
      enabled: true
      factors:
        - "similarity_score"
        - "authority_level"
        - "recency"

# WebSearch 配置 - 扩展支持法律搜索
web_search:
  providers:
    - id: "duckduckgo"
      name: "DuckDuckGo"
      free: true
      requires_api_key: false
      description: "DuckDuckGo API"
    
    - id: "perplexity_legal"
      name: "Perplexity Legal"
      free: false
      requires_api_key: true
      description: "Perplexity.ai 法律专业搜索"
      config:
        api_key: "${PERPLEXITY_API_KEY}"
        model: "sonar-pro"
        legal_prompt_template: |
          你是专业的中国法律案例搜索助手。请根据查询条件搜索相关判例：
          查询条件：%s
          请返回相关判例的详细信息。
    
    - id: "browser_use"
      name: "Browser Use"
      free: true
      requires_api_key: false
      description: "浏览器自动化法律案例搜索"
      config:
        python_path: "python3"
        script_path: "./scripts/browser_use_search.py"
        headless_mode: true
        timeout: 60
  
  default:
    provider: "duckduckgo"
    max_results: 5
    include_date: true
    compression_method: "none"
    blacklist: []
  
  timeout: 10

# 对话服务配置 - 法律领域优化
conversation:
  max_rounds: 5
  keyword_threshold: 0.3
  embedding_top_k: 10
  vector_threshold: 0.5
  rerank_threshold: 0.5
  rerank_top_k: 5
  fallback_strategy: "model"
  
  # 法律专业化提示词
  fallback_prompt: |
    你是一个专业、友好的法律AI助手。请根据你的知识直接回答用户的法律问题。
    
    ## 回复要求
    - 直接回答用户的法律问题
    - 简洁清晰，言之有物
    - 如果涉及具体案件或个人法律建议，建议咨询专业律师
    - 使用专业、准确的法律术语
    
    ## 用户的问题是:
    {{.Query}}
  
  # 指代消解配置
  enable_rewrite: true
  rewrite_prompt_system: |
    你是专注于法律领域指代消解的智能助手。根据历史对话上下文，
    清晰识别用户问题中的代词并替换为明确的法律概念或实体。
    
    ## 改写目标
    - 将"它"、"这个"、"该法条"、"此案"等代词替换为具体的法律实体
    - 补全省略的关键法律信息
    - 保持法律问题的专业性和准确性
    - 改写后的问题字数控制在30字以内
    
    ## 法律领域特殊要求
    - 法条引用必须完整：如"《刑法》第264条"
    - 案件类型要明确：如"盗窃罪"、"合同纠纷"
    - 当事人角色要清晰：如"被告人"、"原告"
  
  # 关键词提取 - 法律专业优化
  keywords_extraction_prompt: |
    你是专业的法律关键词提取助手。从用户的法律问题中提取最重要的法律概念和关键词。
    
    ## 要求
    - 优先提取法律专业术语（罪名、案由、法条等）
    - 识别法律主体（当事人、法院、律师等）
    - 提取法律行为和后果
    - 关键词数量不超过5个
    - 使用逗号分隔
    
    ## 示例
    用户问题：交通肇事罪的量刑标准是什么？
    输出：交通肇事罪, 量刑标准, 刑法, 刑罚, 法定刑
    
    用户问题：{{.Query}}
    输出：

# 知识图谱提取配置 - 法律领域专用
extract:
  # 法律知识图谱提取
  extract_graph:
    description: |
      基于中国法律体系构建统一知识图谱，支持12大法律文档源的跨文档关系提取。
      
      ## 核心目标
      1. 识别文档类型（判决书、法律法规、司法解释等）
      2. 提取法律实体（法条、罪名、当事人、法院等）
      3. 建立跨文档关系（引用、适用、指导等）
      4. 构建完整法律推理链
    
    # 法律关系类型 - 完整版
    tags:
      # 法规体系关系
      - "属于" - "组成部分" - "修订" - "废止" - "替代" - "基于"
      # 司法解释关系
      - "解释" - "澄清" - "细化"
      # 判决书关系
      - "引用" - "适用" - "参照" - "援引"
      # 指导性案例关系
      - "指导" - "参考" - "发布" - "示范" - "阐释"
      # 案件实体关系
      - "被指控" - "被起诉" - "判处" - "承担责任" - "代理" - "主审" - "公诉"
      # 证据关系
      - "证明" - "支持" - "矛盾"
      # 案件关联关系
      - "类似案例" - "引用判例" - "上诉" - "基于指导性案例"
      # 概念关系
      - "定义" - "包含" - "涉及概念"
      # 组织文档关系
      - "发文" - "审理"
      # 立法解释关系（P0优先级）
      - "立法解释" - "授权解释"
      # 法律批复关系（P0优先级）
      - "批复" - "指导适用" - "约束"
      # 国际条约关系
      - "实施" - "加入" - "优先"
      # 仲裁关系
      - "仲裁" - "执行" - "撤销仲裁"
      # 行政执法关系
      - "处罚" - "基于法规" - "被质疑" - "维持" - "撤销"
      # 检察监督关系
      - "监督" - "抗诉" - "启动再审"
      # 劳动争议关系
      - "劳动仲裁前置" - "劳动争议"
      # 地方法规关系
      - "地方实施" - "区域适用"

# 租户配置
tenant:
  enable_cross_tenant_access: true
  
  # 资源租户配置
  resource_tenant:
    enable_isolation: true
    audit_cross_tenant_access: true
    max_shared_knowledge_bases: 1000
  
  # 租户权限配置
  permissions:
    system_admin:
      can_access_all_tenants: true
      can_create_global_kb: true
      can_manage_sharing: true
    
    tenant_admin:
      can_create_tenant_kb: true
      can_manage_tenant_sharing: true
      can_view_tenant_audit: true
    
    user:
      can_create_private_kb: true
      can_share_private_kb: true
      max_private_kb: 100

# 安全配置
security:
  # 数据脱敏
  data_masking:
    enabled: true
    patterns:
      - name: "身份证号"
        regex: '\d{15}|\d{18}'
        replacement: "***"
      - name: "手机号"
        regex: '1[3-9]\d{9}'
        replacement: "***"
      - name: "银行卡号"
        regex: '\d{4}-\d{4}-\d{4}-\d{4}'
        replacement: "****-****-****-****"
  
  # API限流
  rate_limiting:
    enabled: true
    requests_per_minute: 100
    burst_size: 20
  
  # 审计日志
  audit:
    enabled: true
    log_level: "INFO"
    retention_days: 2555  # 7年
    include_request_body: true
    include_response_body: false

# 性能配置
performance:
  # 缓存配置
  cache:
    redis:
      enabled: true
      host: "localhost"
      port: 6379
      db: 0
      password: "${REDIS_PASSWORD}"
      max_connections: 100
    
    # 分层缓存
    layers:
      permission_cache:
        ttl: 300s
        max_size: 10000
      
      search_cache:
        ttl: 1800s  # 30分钟
        max_size: 5000
      
      external_search_cache:
        ttl: 3600s  # 1小时
        max_size: 1000
  
  # 数据库连接池
  database:
    max_open_connections: 100
    max_idle_connections: 10
    connection_max_lifetime: 3600s
  
  # 并发控制
  concurrency:
    max_concurrent_searches: 50
    max_concurrent_extractions: 10
    max_concurrent_external_searches: 5

# 监控配置
monitoring:
  # 指标收集
  metrics:
    enabled: true
    endpoint: "/metrics"
    
    # 自定义指标
    custom_metrics:
      - name: "legal_document_processed"
        type: "counter"
        description: "法律文档处理数量"
      
      - name: "external_case_search_duration"
        type: "histogram"
        description: "外部判例搜索耗时"
      
      - name: "permission_check_duration"
        type: "histogram"
        description: "权限检查耗时"
  
  # 健康检查
  health:
    enabled: true
    endpoint: "/health"
    checks:
      - name: "database"
        timeout: 5s
      - name: "redis"
        timeout: 3s
      - name: "neo4j"
        timeout: 5s

# 日志配置
logging:
  level: "INFO"
  format: "json"
  
  # 日志轮转
  rotation:
    max_size: "100MB"
    max_age: 30
    max_backups: 10
  
  # 结构化日志字段
  fields:
    service: "aiplusall-kb"
    version: "2.0.0"
    environment: "${ENVIRONMENT}"
```

### 配置文件验证规则

1. **必需配置项检查**
   - 数据库连接配置
   - 缓存配置
   - 权限模型配置
   - 法律专业化配置

2. **配置一致性检查**
   - 图谱引擎配置与数据库配置一致
   - 权限配置与租户配置一致
   - 外部搜索配置与API密钥配置一致

3. **性能配置验证**
   - 缓存TTL设置合理
   - 连接池大小适当
   - 并发限制合理

4. **安全配置验证**
   - 敏感信息使用环境变量
   - 审计日志配置完整
   - 数据脱敏规则覆盖全面

## 测试策略

### 单元测试
- 权限计算逻辑测试
- 法律元数据提取测试
- 图谱引擎切换逻辑测试
- 数据模型验证测试

### 集成测试
- 跨租户访问流程测试
- 法律检索端到端测试
- 分享功能完整流程测试
- 图谱引擎集成测试

### 属性测试
- 权限隔离属性测试（最少100次迭代）
- 资源租户一致性测试（最少100次迭代）
- 分享时效性测试（最少100次迭代）
- 审计完整性测试（最少100次迭代）

### 性能测试
- 权限查询性能测试（目标<100ms）
- 大规模知识库列表查询测试
- 法律检索性能测试（目标<2s）
- 并发访问压力测试

### 安全测试
- 权限绕过测试
- 跨租户数据泄露测试
- SQL注入和XSS测试
- 敏感信息泄露测试