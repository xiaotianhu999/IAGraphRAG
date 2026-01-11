# WeKnora 知识库改造完整方案

> 目标：在不破坏现有租户隔离与主链路的前提下，支持：
> 1）全租户共享知识库；2）租户私有知识库；3）用户私有知识库；4）用户私有知识库可分享给他人。
> 
> 特别说明：知识图谱功能仅对全租户和租户知识库有效，用户私有知识库不构建知识图谱。

## 目录

1. [现状盘点](#1-现状盘点)
2. [需求澄清与目标定义](#2-需求澄清与目标定义)
3. [总体设计](#3-总体设计)
4. [数据模型设计](#4-数据模型设计)
5. [权限模型](#5-权限模型)
6. [查询与检索策略](#6-查询与检索策略)
7. [API设计](#7-api设计)
8. [知识图谱技术对比与集成](#8-知识图谱技术对比与集成)
9. [迁移与兼容策略](#9-迁移与兼容策略)
10. [风险点与对策](#10-风险点与对策)
11. [实施清单](#11-实施清单)
12. [总结](#12-总结)

---

## 1. 现状盘点

### 1.1 数据模型现状

**KnowledgeBase**：`internal/types/knowledgebase.go`
- 核心字段：`ID/Name/Type/Description/TenantID/IsTemporary`
- 配置字段：`ChunkingConfig/VLMConfig/ExtractConfig/FAQConfig/QuestionGenerationConfig`
- 模型字段：`EmbeddingModelID/SummaryModelID`
- **租户隔离依赖字段：`TenantID`**（强隔离主键）

**Knowledge**：`internal/types/knowledge.go`
- 标识字段：`ID/TenantID/KnowledgeBaseID/TagID`
- 状态字段：`ParseStatus/SummaryStatus/EnableStatus`
- 类型字段：`Type`（manual/faq）
- 内容字段：`Title/Description/Source/FileHash`

**Chunk**：`internal/types/chunk.go`
- 标识字段：`ID/TenantID/KnowledgeID/KnowledgeBaseID`
- 类型字段：`ChunkType`（text/image_ocr/image_caption/summary/entity/relationship/faq/web_search）
- 状态字段：`Status/Flags`（推荐状态等位标志）
- 内容字段：`Content/ContentHash/ImageInfo`

**User**：`internal/types/user.go`
- 标识字段：`ID/Username/Email/TenantID`
- 权限字段：`Role`（admin/user）、`CanAccessAllTenants`（超级管理员标识）
- 配置字段：`MenuConfig`（用户级菜单权限）

**结论**：当前所有数据（KB/Knowledge/Chunk）都以 `tenant_id` 为强隔离主键之一，用户权限模型简单（仅admin/user两级）。
### 1.2 鉴权与租户注入现状

**认证中间件**：`internal/middleware/auth.go`

**认证方式**：
1. **JWT Token认证**（优先）：`Authorization: Bearer <token>`
2. **API-Key认证**（备选）：`X-API-Key: <api_key>`

**Context注入**：
- `types.TenantIDContextKey`：当前请求租户ID
- `types.TenantInfoContextKey`：租户配置对象（含RetrieverEngines）
- `"user"`：当前用户对象

**跨租户访问控制**：
```go
// canAccessTenant函数的逻辑
1. 目标租户 == 用户租户 → 允许
2. 用户.CanAccessAllTenants == true → 允许（超级管理员）
3. cfg.Tenant.EnableCrossTenantAccess == true → 允许（全局开关，默认关闭）
4. 其他情况 → 拒绝（403 Forbidden）
```

**租户隔离机制**：
- 数据库层：`internal/database/tenant_isolation.go` 的GORM中间件
- 自动在Query/Create/Update/Delete前添加`tenant_id`条件
- 超级管理员可通过`skip_tenant_isolation`标记绕过隔离

**结论**：系统默认强租户隔离，仅超级管理员或开启全局开关后可切租户。

### 1.3 Handler层强租户校验

**知识库Handler**：`internal/handler/knowledgebase.go`

`validateAndGetKnowledgeBase`函数：
```go
// 1. 从Context获取当前租户ID
tenantID := c.Get(types.TenantIDContextKey.String())

// 2. 从URL获取知识库ID
id := c.Param("id")

// 3. 从数据库查询知识库
kb := h.service.GetKnowledgeBaseByID(ctx, id)

// 4. 强制校验：kb.TenantID == 当前租户ID
if kb.TenantID != tenantID {
    return errors.NewForbiddenError("No permission to operate")
}
```

因此：即便 KB 本身能被 repo 查到（repo 仅 `where id=?`），最终也会被 handler 拦截。

### 1.4 Service/Repository 查询链路现状

- KB 列表：`ListKnowledgeBases` → `ListKnowledgeBasesByTenantID(tenantID)`
  - repo 查询固定条件：`tenant_id = ? AND is_temporary = false`

- KB 统计：knowledge/chunk 计数接口均接收 `tenantID` 参数（来自当前 request context）

- HybridSearch：`KnowledgeBaseService.HybridSearch`
  - 通过 `ctx.Value(TenantInfoContextKey)` 取"当前租户"的 RetrieverEngines
  - 然后在向量/关键词检索引擎上执行检索

**关键结论**：
- **存储/检索引擎选择是"按 tenantInfo（当前租户）"决定的**。
- 如果要让租户 B 访问租户 A 创建的 KB，当前实现会：
  - handler 阶段直接拒绝（租户不一致）
  - 即便放开 handler，HybridSearch 也会使用租户 B 的引擎配置去查租户 A 的索引，极可能查不到或查错。

因此：要支持"全租户共享/跨租户分享"，必须引入"资源所属租户（ResourceTenant）"的概念，并在检索/统计/删除等操作中使用 KB 的资源租户配置，而不是请求租户配置。

---

## 2. 需求澄清与目标定义

### 2.1 四个核心诉求

1. **全租户共享知识库（Global Shared）**
   - 所有租户可读
   - 写/维护权限可控（通常只允许系统管理员或指定维护者）

2. **租户私有知识库（Tenant Private）**
   - 仅该租户成员可见（可进一步限定只允许管理员维护）

3. **用户私有知识库（User Private）**
   - 默认仅创建者本人可见

4. **用户可将自己的私有知识库共享给他人（Share）**
   - 共享给指定用户（可扩展共享给租户）
   - 共享权限：只读 / 可编辑 / 可管理（可再次分享）
   - 支持撤销与可选过期

### 2.2 方案必须满足的工程约束

- **向后兼容**：
  - 不破坏现有 `tenant_id` 隔离语义（默认仍按 tenant 私有工作）
  - 不改动已有核心类型字段含义（尽量新增字段/表，而不是替换）

- **可落地**：
  - 明确 DB 迁移策略
  - 明确权限判断与查询策略
  - 明确对现有 handler/service 的最小改造点

---

## 3. 总体设计

### 3.1 核心概念：三层"所有权 + 可见性"模型

本方案把"谁拥有（Owner）"与"谁可见（Visibility）"分离：

- **Owner（所有者）**决定"默认管理权/计费归属/底层存储与检索引擎归属"。
- **Visibility（可见性）**决定"可读范围"。
- **Share（分享）**决定"非所有者的扩展权限"。

### 3.2 三类知识库

1) **全局共享 KB**
- OwnerType = system
- ResourceTenantID = SYSTEM（建议独立系统租户或全局引擎配置）
- Visibility = public

2) **租户私有 KB**
- OwnerType = tenant
- ResourceTenantID = 该租户
- Visibility = private 或 shared（共享给指定用户/租户）

3) **用户私有 KB**
- OwnerType = user
- OwnerUserID = 创建者
- ResourceTenantID = 创建者所属租户（用于文件/向量/索引的落库归属）
- Visibility = private（默认）
- 可通过 Share 变为共享（Visibility=shared）

> 说明：即便是"用户私有 KB"，底层依旧需要一个资源租户来承载存储与检索引擎配置。

---

## 4. 数据模型设计

### 4.1 扩展 KnowledgeBase 表

在现有 `knowledge_bases`（对应 `types.KnowledgeBase`）上新增字段：

```go
// 建议新增枚举
// OwnerType: system | tenant | user
// Visibility: private | shared | public

type KnowledgeBase struct {
    // 现有字段...
    ID       string
    TenantID uint64 // 现有字段：保留，但语义升级为 ResourceTenantID

    // === 新增字段 ===
    OwnerType     string  `gorm:"type:varchar(16);default:'tenant';index"`   // system/tenant/user
    OwnerTenantID uint64  `gorm:"index"`                                      // 所有者租户（多数情况下 = TenantID）
    OwnerUserID   *string `gorm:"type:varchar(36);index"`                    // 用户私有 KB 才有值

    Visibility string `gorm:"type:varchar(16);default:'private';index"`      // private/shared/public

    // 可选：用于 UI 或审计显示来源
    Source string `gorm:"type:varchar(32);default:''"` // e.g. system/user/tenant
}
```

**关键约定**：
- **TenantID 保留，但语义升级为 ResourceTenantID**：
  - 用于承载向量/关键词索引、文件存储配额、知识图谱数据等"资源归属"。
  - 解决跨租户访问时"请求租户"和"资源租户"不一致的问题。

- `OwnerTenantID` 与 `OwnerUserID` 用于表达所有权：
  - tenant 私有：OwnerType=tenant，OwnerTenantID=TenantID，OwnerUserID=nil
  - user 私有：OwnerType=user，OwnerTenantID=TenantID（=创建者租户），OwnerUserID=创建者
  - global：OwnerType=system（OwnerTenantID 可为 0 或 SYSTEM_TENANT_ID），OwnerUserID=nil

### 4.2 新增 KnowledgeBaseShare 表

用于记录分享关系（支持用户私有 KB 分享给他人，也可支持租户级共享）。

```go
type KnowledgeBaseShare struct {
    ID              string `gorm:"type:varchar(36);primaryKey"`
    KnowledgeBaseID string `gorm:"type:varchar(36);index"`

    SharedByUserID string `gorm:"type:varchar(36);index"`

    // 分享目标类型：user | tenant
    TargetType string `gorm:"type:varchar(16);index"`

    // 目标标识
    TargetUserID   *string `gorm:"type:varchar(36);index"`
    TargetTenantID *uint64 `gorm:"index"`

    // 权限：read | write | admin
    Permission string `gorm:"type:varchar(16);default:'read';index"`

    ExpiresAt  *time.Time `gorm:"index"`
    IsEnabled  bool      `gorm:"default:true;index"`

    CreatedAt time.Time `gorm:"index"`
    UpdatedAt time.Time
}
```

### 4.3 可选：访问审计表

```go
type KnowledgeBaseAccessLog struct {
    ID              string    `gorm:"type:varchar(36);primaryKey"`
    KnowledgeBaseID string    `gorm:"type:varchar(36);index"`
    UserID          string    `gorm:"type:varchar(36);index"`
    TenantID        uint64    `gorm:"index"`
    Action          string    `gorm:"type:varchar(32);index"` // read/write/delete/share
    IPAddress       string    `gorm:"type:varchar(45)"`
    UserAgent       string    `gorm:"type:text"`
    CreatedAt       time.Time `gorm:"index"`
}
```

---

## 5. 权限模型

### 5.1 基本角色与现状复用

当前系统已有：
- 超级管理员：`CanAccessAllTenants=true`
- 租户管理员：`Role=admin`
- 普通用户：`Role=user`

并且用户有 `MenuConfig`（功能权限）与租户 `MenuConfig`。

### 5.2 新增"知识库级权限"的计算规则

引入统一权限计算：`PermissionLevel`：
- `none | read | write | admin | owner | system_admin`

优先级（从高到低）：
1. 超级管理员：system_admin
2. 知识库所有者：owner
   - OwnerType=tenant：同租户管理员/允许的 tenant 成员（由产品决定）
   - OwnerType=user：OwnerUserID == currentUser
3. 分享权限：admin/write/read（按 share 记录）
4. public：read（全员可读）

### 5.3 行为授权矩阵

| 操作 | system_admin | owner | admin(share) | write(share) | read(share) | public 用户 |
|---|---|---|---|---|---|---|
| 查看 KB | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| HybridSearch | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| 新增知识/导入 | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| 修改 KB 配置 | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |
| 删除 KB | ✅ | ✅ | ❌（可选） | ❌ | ❌ | ❌ |
| 分享/撤销分享 | ✅ | ✅ | ✅ | ❌ | ❌ | ❌ |

### 5.4 权限校验服务实现

```go
// internal/application/service/knowledge_base_permission_service.go
package service

import (
    "context"
    "time"
    
    "github.com/aiplusall/aiplusall-kb/internal/types"
    "github.com/aiplusall/aiplusall-kb/internal/types/interfaces"
)

type KnowledgeBasePermissionService struct {
    kbRepo    interfaces.KnowledgeBaseRepository
    shareRepo interfaces.KnowledgeBaseShareRepository
    userRepo  interfaces.UserRepository
}

type PermissionLevel string

const (
    PermissionNone        PermissionLevel = "none"
    PermissionRead        PermissionLevel = "read"
    PermissionWrite       PermissionLevel = "write"
    PermissionAdmin       PermissionLevel = "admin"
    PermissionOwner       PermissionLevel = "owner"
    PermissionSystemAdmin PermissionLevel = "system_admin"
)

func (s *KnowledgeBasePermissionService) CheckPermission(
    ctx context.Context,
    userID string,
    kbID string,
) (PermissionLevel, error) {
    // 1. 获取用户信息
    user, err := s.userRepo.GetByID(ctx, userID)
    if err != nil {
        return PermissionNone, err
    }
    
    // 2. 超级管理员检查
    if user.CanAccessAllTenants {
        return PermissionSystemAdmin, nil
    }
    
    // 3. 获取知识库信息
    kb, err := s.kbRepo.GetByID(ctx, kbID)
    if err != nil {
        return PermissionNone, err
    }
    
    // 4. 所有者检查
    if s.isOwner(user, kb) {
        return PermissionOwner, nil
    }
    
    // 5. 分享权限检查
    sharePermission, err := s.getSharePermission(ctx, userID, user.TenantID, kbID)
    if err == nil && sharePermission != PermissionNone {
        return sharePermission, nil
    }
    
    // 6. 公开知识库检查
    if kb.Visibility == "public" {
        return PermissionRead, nil
    }
    
    return PermissionNone, nil
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
) (PermissionLevel, error) {
    shares, err := s.shareRepo.GetByKnowledgeBaseID(ctx, kbID)
    if err != nil {
        return PermissionNone, err
    }
    
    now := time.Now()
    for _, share := range shares {
        // 检查是否过期和启用
        if !share.IsEnabled || (share.ExpiresAt != nil && share.ExpiresAt.Before(now)) {
            continue
        }
        
        // 检查目标匹配
        if share.TargetType == "user" && share.TargetUserID != nil && *share.TargetUserID == userID {
            return PermissionLevel(share.Permission), nil
        }
        if share.TargetType == "tenant" && share.TargetTenantID != nil && *share.TargetTenantID == tenantID {
            return PermissionLevel(share.Permission), nil
        }
    }
    
    return PermissionNone, nil
}
```

---

## 6. 查询与检索策略

### 6.1 可访问知识库列表（List）

返回"当前用户可见"的 KB：
- A. Tenant 私有 KB：OwnerType=tenant AND OwnerTenantID=currentTenant
- B. User 私有 KB：OwnerType=user AND OwnerUserID=currentUser
- C. 分享给我（user）：share.TargetUserID=currentUser 且未过期且启用
- D. 分享给我所在租户（tenant，可选）：share.TargetTenantID=currentTenant
- E. 全局 public KB：OwnerType=system AND Visibility=public

```go
func (s *KnowledgeBaseService) ListAccessibleKnowledgeBases(
    ctx context.Context,
    userID string,
    tenantID uint64,
    params *ListParams,
) ([]*types.KnowledgeBase, error) {
    // 构建查询条件
    conditions := []string{}
    args := []interface{}{}
    
    // A. 租户私有
    conditions = append(conditions, "(owner_type = 'tenant' AND owner_tenant_id = ?)")
    args = append(args, tenantID)
    
    // B. 用户私有
    conditions = append(conditions, "(owner_type = 'user' AND owner_user_id = ?)")
    args = append(args, userID)
    
    // C. 分享给用户
    conditions = append(conditions, `(id IN (
        SELECT knowledge_base_id FROM knowledge_base_shares 
        WHERE target_type = 'user' AND target_user_id = ? 
        AND is_enabled = true AND (expires_at IS NULL OR expires_at > NOW())
    ))`)
    args = append(args, userID)
    
    // D. 分享给租户
    conditions = append(conditions, `(id IN (
        SELECT knowledge_base_id FROM knowledge_base_shares 
        WHERE target_type = 'tenant' AND target_tenant_id = ? 
        AND is_enabled = true AND (expires_at IS NULL OR expires_at > NOW())
    ))`)
    args = append(args, tenantID)
    
    // E. 全局公开
    conditions = append(conditions, "(owner_type = 'system' AND visibility = 'public')")
    
    query := fmt.Sprintf("WHERE (%s) AND is_temporary = false", 
        strings.Join(conditions, " OR "))
    
    return s.kbRepo.ListWithConditions(ctx, query, args, params)
}
```

### 6.2 HybridSearch（检索引擎选择）

现状：HybridSearch 使用 `ctx` 中的 `TenantInfo` 选择引擎。

改造建议：
```go
func (s *KnowledgeBaseService) HybridSearch(
    ctx context.Context,
    kbID string,
    params *SearchParams,
) (*SearchResult, error) {
    // 1. 权限校验
    userID := getUserIDFromContext(ctx)
    permission, err := s.permissionService.CheckPermission(ctx, userID, kbID)
    if err != nil || permission == PermissionNone {
        return nil, errors.NewForbiddenError("No permission to search")
    }
    
    // 2. 获取知识库信息
    kb, err := s.kbRepo.GetByID(ctx, kbID)
    if err != nil {
        return nil, err
    }
    
    // 3. 获取资源租户配置（关键改造点）
    resourceTenantInfo, err := s.tenantService.GetTenantInfo(ctx, kb.TenantID)
    if err != nil {
        return nil, err
    }
    
    // 4. 使用资源租户的引擎配置执行检索
    retrieveEngine := resourceTenantInfo.GetEffectiveEngines()
    return s.executeSearch(ctx, retrieveEngine, kbID, params)
}
```

### 6.3 统计/删除等"深度操作"同样需要资源租户

例如 DeleteKnowledgeBase 当前用 request tenant 来：
- 读 knowledge 列表
- 删 chunk
- 删向量
- 调整 tenant storage
- 删图谱

共享/跨租户后，必须改为：
```go
func (s *KnowledgeBaseService) DeleteKnowledgeBase(
    ctx context.Context,
    kbID string,
) error {
    // 1. 权限校验
    userID := getUserIDFromContext(ctx)
    permission, err := s.permissionService.CheckPermission(ctx, userID, kbID)
    if err != nil || permission < PermissionOwner {
        return errors.NewForbiddenError("No permission to delete")
    }
    
    // 2. 获取知识库信息
    kb, err := s.kbRepo.GetByID(ctx, kbID)
    if err != nil {
        return err
    }
    
    // 3. 使用资源租户ID进行所有删除操作
    resourceTenantID := kb.TenantID
    
    // 删除知识条目（使用资源租户ID）
    err = s.knowledgeRepo.DeleteByKnowledgeBaseID(ctx, resourceTenantID, kbID)
    if err != nil {
        return err
    }
    
    // 删除向量索引（使用资源租户配置）
    resourceTenantInfo, err := s.tenantService.GetTenantInfo(ctx, resourceTenantID)
    if err != nil {
        return err
    }
    
    vectorEngine := resourceTenantInfo.GetVectorEngine()
    err = vectorEngine.DeleteByKnowledgeBase(ctx, kbID)
    if err != nil {
        return err
    }
    
    // 调整存储配额（计入资源租户）
    err = s.storageService.AdjustQuota(ctx, resourceTenantID, -kb.StorageUsed)
    if err != nil {
        return err
    }
    
    return s.kbRepo.Delete(ctx, kbID)
}
```

---

## 7. API设计

### 7.1 创建知识库

#### 7.1.1 创建租户私有 KB（兼容现有）
`POST /api/v1/knowledge-bases`

- 权限：租户管理员（或你们希望开放给普通用户时，增加功能开关）
- 默认：OwnerType=tenant，Visibility=private

#### 7.1.2 创建用户私有 KB（新增）
`POST /api/v1/my/knowledge-bases`

请求体：
```json
{
  "name": "个人知识库",
  "type": "document",
  "description": "...",
  "visibility": "private"
}
```

服务端写入：
- OwnerType=user
- OwnerUserID=currentUser
- OwnerTenantID=currentTenant
- TenantID(ResourceTenantID)=currentTenant

### 7.2 获取知识库列表

`GET /api/v1/knowledge-bases`

支持参数：
- `scope=all|tenant|user|global`（可选）
- `include_shared=true|false`（默认 true）

响应：
```json
{
  "data": [
    {
      "id": "kb-xxx",
      "name": "...",
      "owner_type": "user",
      "visibility": "shared",
      "my_permission": "write",
      "resource_tenant_id": 123
    }
  ]
}
```

### 7.3 知识库详情/更新/删除

`GET /api/v1/knowledge-bases/:id`
- 由"强租户校验"改为"权限校验（read）"

`PUT /api/v1/knowledge-bases/:id`
`DELETE /api/v1/knowledge-bases/:id`
- 由"管理员 + 强租户校验"改为"权限校验（write/admin/owner）"

### 7.4 分享管理（新增）

#### 创建分享
`POST /api/v1/knowledge-bases/:id/shares`

```json
{
  "shares": [
    {
      "target_type": "user",
      "target_user_id": "user-456",
      "permission": "read",
      "expires_at": "2026-12-31T23:59:59Z"
    }
  ]
}
```

#### 管理分享
`GET /api/v1/knowledge-bases/:id/shares`
`PATCH /api/v1/knowledge-bases/:id/shares/:share_id`
`DELETE /api/v1/knowledge-bases/:id/shares/:share_id`

权限：owner 或 share-admin 或 system_admin。

### 7.5 全局共享 KB（管理端）

`POST /api/v1/system/knowledge-bases`（system_admin）

全局共享 KB 的 ResourceTenantID：
- 方案 A（推荐）：引入系统租户（或 system 配置）承载索引与存储。
- 方案 B：要求所有租户引擎一致并共享同一底层检索存储（约束较强，不推荐）。

---

## 8. 知识图谱技术对比与集成

### 8.1 技术架构对比分析

#### 8.1.1 WeKnora 当前知识图谱实现

**核心架构**：
- **存储引擎**: Neo4j图数据库
- **构建方式**: 基于LLM的实体关系抽取
- **检索策略**: 实体搜索 + 文本块搜索并行
- **数据流程**: 文档分块 → 实体抽取 → 关系构建 → Neo4j存储 → 查询检索

**关键特性**：
```go
// 核心数据结构
type GraphData struct {
    Node     []*GraphNode       // 实体节点
    Relation []*GraphRelation   // 关系边
}

type ExtractConfig struct {
    Enabled   bool             // 知识库级别开关
    Text      string           // 示例文本
    Tags      []string         // 关系类型
    Nodes     []*GraphNode     // 实体类型配置
    Relations []*GraphRelation // 关系类型配置
}
```

**优势**：
- 知识库级别的精细化配置
- 支持异步批量重建
- 与现有RAG系统深度集成
- 支持直接和间接关系查询

**局限性**：
- 依赖Neo4j单一存储
- 实体抽取质量依赖LLM
- 缺乏社区检测和层次化分析
- 图谱构建相对简单

#### 8.1.2 Microsoft GraphRAG

参考链接：https://github.com/microsoft/graphrag

**核心架构**：
- **存储引擎**: 抽象知识模型，支持多种数据库
- **构建方式**: LLM驱动的实体关系抽取 + 社区检测
- **检索策略**: 全局摘要 + 本地检索的层次化方法
- **数据流程**: 文档准备 → 分块 → 图谱抽取 → 社区检测 → 报告生成 → 嵌入

**核心创新**：
- **社区检测**: 使用图机器学习算法进行语义聚合
- **层次化分析**: 多级摘要支持抽象问题回答
- **全局查询**: 能回答跨文档的高层次问题
- **LLM缓存**: 提高重复查询效率

**优势**：
- 强大的全局理解能力
- 支持抽象和总结性查询
- 工业级的可扩展性
- 完善的缓存机制

**局限性**：
- 复杂度高，资源消耗大
- 构建时间长
- 对LLM质量要求极高
- 配置复杂

#### 8.1.3 HKUDS LightRAG

参考链接：https://github.com/HKUDS/LightRAG

**核心架构**：
- **存储引擎**: 多存储支持(Neo4j/PostgreSQL/MongoDB等)
- **构建方式**: 双层检索系统(低层+高层知识发现)
- **检索策略**: 图结构增强的文本索引和检索
- **数据流程**: 文档分块 → 实体关系抽取 → 图谱构建 → 双层检索

**核心创新**：
- **双层检索**: 结合向量相似性和图结构
- **多模态支持**: 通过RAG-Anything集成
- **轻量化设计**: 相比GraphRAG更简单快速
- **灵活存储**: 支持多种数据库后端

**优势**：
- 简单快速的部署
- 良好的性能表现
- 多存储后端支持
- 活跃的社区开发

**局限性**：
- 相对较新，生态不够成熟
- 文档和最佳实践有限
- 高级功能仍在开发中

### 8.2 集成解决方案设计

#### 8.2.1 整体架构设计

```go
// 知识图谱引擎接口
type KnowledgeGraphEngine interface {
    // 构建图谱
    BuildGraph(ctx context.Context, chunks []*Chunk, config *GraphConfig) error
    
    // 查询图谱
    QueryGraph(ctx context.Context, query string, params *QueryParams) (*GraphResult, error)
    
    // 重建图谱
    RebuildGraph(ctx context.Context, kbID string, batchSize int) error
    
    // 获取引擎信息
    GetEngineInfo() *EngineInfo
}

// 图谱引擎类型
type GraphEngineType string

const (
    EngineWeKnora   GraphEngineType = "weknora"
    EngineGraphRAG  GraphEngineType = "graphrag"
    EngineLightRAG  GraphEngineType = "lightrag"
)

// 图谱配置
type GraphConfig struct {
    EngineType     GraphEngineType `json:"engine_type"`
    ModelConfig    *ModelConfig    `json:"model_config"`
    StorageConfig  *StorageConfig  `json:"storage_config"`
    ExtractConfig  *ExtractConfig  `json:"extract_config"`
    EngineSpecific map[string]interface{} `json:"engine_specific"`
}
```

#### 8.2.2 引擎管理器实现

```go
// internal/application/service/graph_engine_manager.go
type GraphEngineManager struct {
    engines map[types.GraphEngineType]types.KnowledgeGraphEngine
    config  *config.Config
    mutex   sync.RWMutex
}

func (m *GraphEngineManager) RegisterEngine(engineType types.GraphEngineType, engine types.KnowledgeGraphEngine) {
    m.mutex.Lock()
    defer m.mutex.Unlock()
    m.engines[engineType] = engine
}

func (m *GraphEngineManager) GetEngine(engineType types.GraphEngineType) (types.KnowledgeGraphEngine, error) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    engine, exists := m.engines[engineType]
    if !exists {
        return nil, fmt.Errorf("engine type %s not registered", engineType)
    }
    return engine, nil
}
```

#### 8.2.3 配置管理

```go
// 图谱引擎配置
type GraphEngineConfig struct {
    DefaultEngine GraphEngineType            `yaml:"default_engine" json:"default_engine"`
    Engines       map[string]*EngineConfig   `yaml:"engines" json:"engines"`
}

type EngineConfig struct {
    Type          GraphEngineType `yaml:"type" json:"type"`
    Enabled       bool           `yaml:"enabled" json:"enabled"`
    WorkingDir    string         `yaml:"working_dir" json:"working_dir"`
    PythonPath    string         `yaml:"python_path" json:"python_path"`
    ServerURL     string         `yaml:"server_url" json:"server_url"`
    ModelConfig   *ModelConfig   `yaml:"model_config" json:"model_config"`
    StorageConfig *StorageConfig `yaml:"storage_config" json:"storage_config"`
    Specific      map[string]interface{} `yaml:"specific" json:"specific"`
}
```

### 8.3 引擎选择指南

#### 8.3.1 WeKnora引擎
**适用场景**:
- 现有系统的平滑升级
- 对响应速度要求高的场景
- 资源有限的环境
- 需要与现有Neo4j基础设施集成

#### 8.3.2 GraphRAG引擎
**适用场景**:
- 需要回答高层次抽象问题
- 大规模文档集合分析
- 科研文献处理
- 对准确性要求极高的场景

#### 8.3.3 LightRAG引擎
**适用场景**:
- 快速原型开发
- 多模态文档处理
- 需要灵活存储选择
- 中等规模的知识库

### 8.4 知识图谱与知识库类型的关系

**重要约束**：
- **全租户共享知识库**：支持知识图谱，使用系统级图谱引擎配置
- **租户私有知识库**：支持知识图谱，使用租户级图谱引擎配置
- **用户私有知识库**：**不支持知识图谱**，仅支持向量检索和关键词检索

这个约束的原因：
1. **资源考虑**：知识图谱构建消耗大量计算资源，用户私有KB通常规模较小，ROI不高
2. **复杂度控制**：避免为每个用户维护独立的图谱存储和检索引擎
3. **成本控制**：减少Neo4j等图数据库的实例数量和维护成本

---

## 9. 迁移与兼容策略

### 9.1 数据迁移（必须）

#### 9.1.1 数据库结构迁移
```sql
-- 1. 扩展 knowledge_bases 表
ALTER TABLE knowledge_bases 
ADD COLUMN owner_type VARCHAR(16) DEFAULT 'tenant',
ADD COLUMN owner_tenant_id BIGINT UNSIGNED,
ADD COLUMN owner_user_id VARCHAR(36),
ADD COLUMN visibility VARCHAR(16) DEFAULT 'private',
ADD COLUMN source VARCHAR(32) DEFAULT '';

-- 2. 添加索引
CREATE INDEX idx_kb_owner_type ON knowledge_bases(owner_type);
CREATE INDEX idx_kb_owner_tenant_id ON knowledge_bases(owner_tenant_id);
CREATE INDEX idx_kb_owner_user_id ON knowledge_bases(owner_user_id);
CREATE INDEX idx_kb_visibility ON knowledge_bases(visibility);

-- 3. 创建分享表
CREATE TABLE knowledge_base_shares (
    id VARCHAR(36) PRIMARY KEY,
    knowledge_base_id VARCHAR(36) NOT NULL,
    shared_by_user_id VARCHAR(36) NOT NULL,
    target_type VARCHAR(16) NOT NULL,
    target_user_id VARCHAR(36),
    target_tenant_id BIGINT UNSIGNED,
    permission VARCHAR(16) DEFAULT 'read',
    expires_at TIMESTAMP NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_kbs_kb_id (knowledge_base_id),
    INDEX idx_kbs_shared_by (shared_by_user_id),
    INDEX idx_kbs_target_type (target_type),
    INDEX idx_kbs_target_user (target_user_id),
    INDEX idx_kbs_target_tenant (target_tenant_id),
    INDEX idx_kbs_permission (permission),
    INDEX idx_kbs_expires_at (expires_at),
    INDEX idx_kbs_enabled (is_enabled)
);

-- 4. 创建访问审计表（可选）
CREATE TABLE knowledge_base_access_logs (
    id VARCHAR(36) PRIMARY KEY,
    knowledge_base_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    tenant_id BIGINT UNSIGNED NOT NULL,
    action VARCHAR(32) NOT NULL,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_kbal_kb_id (knowledge_base_id),
    INDEX idx_kbal_user_id (user_id),
    INDEX idx_kbal_tenant_id (tenant_id),
    INDEX idx_kbal_action (action),
    INDEX idx_kbal_created_at (created_at)
);
```

#### 9.1.2 历史数据回填
```sql
-- 回填历史知识库数据
UPDATE knowledge_bases SET 
    owner_type = 'tenant',
    owner_tenant_id = tenant_id,
    owner_user_id = NULL,
    visibility = 'private'
WHERE owner_type IS NULL;
```

### 9.2 代码兼容（必须）

#### 9.2.1 Handler层改造
```go
// 替换强租户校验为权限校验
func (h *KnowledgeBaseHandler) validateAndGetKnowledgeBase(c *gin.Context) (*types.KnowledgeBase, error) {
    kbID := c.Param("id")
    userID := getUserIDFromContext(c)
    
    // 权限校验（替代原有的租户校验）
    permission, err := h.permissionService.CheckPermission(c.Request.Context(), userID, kbID)
    if err != nil {
        return nil, err
    }
    
    if permission == PermissionNone {
        return nil, errors.NewForbiddenError("No permission to access this knowledge base")
    }
    
    // 获取知识库信息
    kb, err := h.service.GetKnowledgeBaseByID(c.Request.Context(), kbID)
    if err != nil {
        return nil, err
    }
    
    // 在context中设置权限级别，供后续使用
    c.Set("kb_permission", permission)
    c.Set("resource_tenant_id", kb.TenantID)
    
    return kb, nil
}
```

#### 9.2.2 Service层改造
```go
// HybridSearch 使用资源租户配置
func (s *KnowledgeBaseService) HybridSearch(ctx context.Context, kbID string, params *SearchParams) (*SearchResult, error) {
    // 获取知识库信息
    kb, err := s.kbRepo.GetByID(ctx, kbID)
    if err != nil {
        return nil, err
    }
    
    // 使用资源租户配置（关键改造点）
    resourceTenantInfo, err := s.tenantService.GetTenantInfo(ctx, kb.TenantID)
    if err != nil {
        return nil, err
    }
    
    // 构建检索引擎
    retrieveEngine := resourceTenantInfo.GetEffectiveEngines()
    
    // 执行检索
    return s.executeSearch(ctx, retrieveEngine, kbID, params)
}
```

### 9.3 灰度策略

#### 9.3.1 阶段性上线
1. **Phase 1**: 数据库迁移 + 基础权限服务（保持现有功能不变）
2. **Phase 2**: 用户私有知识库功能
3. **Phase 3**: 知识库分享功能（同租户内）
4. **Phase 4**: 跨租户分享 + 全局知识库
5. **Phase 5**: 知识图谱引擎集成

#### 9.3.2 功能开关控制
```go
type FeatureFlags struct {
    EnableUserPrivateKB     bool `yaml:"enable_user_private_kb"`
    EnableKBSharing         bool `yaml:"enable_kb_sharing"`
    EnableCrossTenantShare  bool `yaml:"enable_cross_tenant_share"`
    EnableGlobalKB          bool `yaml:"enable_global_kb"`
    EnableGraphEngineSwitch bool `yaml:"enable_graph_engine_switch"`
}
```

---

## 10. 风险点与对策

### 10.1 技术风险

#### 10.1.1 检索引擎按租户配置选择
**风险**：跨租户共享 KB 时无法检索
**对策**：引入 resourceTenantInfo（按 kb.TenantID 获取）

#### 10.1.2 存储配额与资源归属
**风险**：共享 KB 的存储算到谁？
**对策**：统一计入 resource tenant（KB 所属资源租户）；全局 KB 计入 system tenant。

#### 10.1.3 API-Key 模式下没有 user 信息
**风险**：用户私有与分享能力需要 user
**对策**：
- API-Key 模式仅允许 tenant 级私有 KB（维持现状），或
- 扩展 API-Key 绑定 user（成本较高）。

### 10.2 性能风险

#### 10.2.1 权限查询性能
**风险**：每次访问都需要查询权限，可能影响性能
**对策**：
- 按（userID, kbID）缓存 permission level
- 设置合理的缓存过期时间
- share 变更时及时失效相关缓存

#### 10.2.2 知识库列表查询复杂化
**风险**：多表JOIN查询可能影响列表性能
**对策**：
- 优化SQL查询，添加必要索引
- 考虑引入搜索引擎（如Elasticsearch）
- 实现分页和过滤优化

### 10.3 业务风险

#### 10.3.1 数据安全与隐私
**风险**：跨租户分享可能导致数据泄露
**对策**：
- 完善的访问审计日志
- 分享权限的细粒度控制
- 定期权限审查机制

#### 10.3.2 资源滥用
**风险**：用户可能创建大量私有知识库
**对策**：
- 设置用户私有知识库数量限制
- 实施存储配额管理
- 监控异常使用行为

---

## 11. 实施清单

### 11.1 Phase 1：基础架构（2-3周）

#### 数据模型与迁移
- [ ] 设计并实现数据库迁移脚本
- [ ] 扩展 KnowledgeBase 结构体
- [ ] 创建 KnowledgeBaseShare 表和对应的 Go 结构体
- [ ] 创建访问审计表（可选）
- [ ] 执行历史数据回填

#### 权限服务
- [ ] 实现 KnowledgeBasePermissionService
- [ ] 定义权限级别枚举和计算逻辑
- [ ] 实现权限缓存机制
- [ ] 编写权限服务单元测试

### 11.2 Phase 2：核心功能改造（3-4周）

#### Handler层改造
- [ ] 替换 validateAndGetKnowledgeBase 中的强租户校验
- [ ] 更新所有知识库相关的 Handler 方法
- [ ] 实现新的权限检查中间件
- [ ] 添加资源租户ID到请求上下文

#### Service层改造
- [ ] 改造 HybridSearch 使用资源租户配置
- [ ] 更新知识库删除逻辑使用资源租户
- [ ] 改造统计接口使用资源租户
- [ ] 实现可访问知识库列表查询

#### Repository层扩展
- [ ] 实现 KnowledgeBaseShareRepository
- [ ] 扩展 KnowledgeBaseRepository 支持新的查询条件
- [ ] 优化查询性能，添加必要索引

### 11.3 Phase 3：用户私有知识库（2-3周）

#### API接口
- [ ] 实现 `POST /api/v1/my/knowledge-bases`
- [ ] 实现 `GET /api/v1/my/knowledge-bases`
- [ ] 更新现有知识库接口支持用户私有类型

#### 前端界面
- [ ] 添加"我的知识库"页面
- [ ] 在知识库列表中区分不同类型
- [ ] 实现知识库类型切换界面

#### 业务逻辑
- [ ] 实现用户私有知识库创建逻辑
- [ ] 确保用户私有知识库不构建知识图谱
- [ ] 实现用户私有知识库的权限控制

### 11.4 Phase 4：分享功能（3-4周）

#### 分享管理API
- [ ] 实现 `POST /api/v1/knowledge-bases/:id/shares`
- [ ] 实现 `GET /api/v1/knowledge-bases/:id/shares`
- [ ] 实现 `PATCH /api/v1/knowledge-bases/:id/shares/:share_id`
- [ ] 实现 `DELETE /api/v1/knowledge-bases/:id/shares/:share_id`

#### 分享业务逻辑
- [ ] 实现分享创建和管理逻辑
- [ ] 实现分享过期检查机制
- [ ] 实现分享权限验证
- [ ] 添加分享操作审计日志

#### 前端分享界面
- [ ] 实现知识库分享对话框
- [ ] 实现分享管理页面
- [ ] 实现分享状态显示
- [ ] 实现分享链接生成（可选）

### 11.5 Phase 5：全局知识库（2-3周）

#### 系统管理功能
- [ ] 实现 `POST /api/v1/system/knowledge-bases`
- [ ] 设计系统租户或全局配置方案
- [ ] 实现全局知识库的特殊权限控制

#### 全局知识库管理
- [ ] 实现系统管理员界面
- [ ] 实现全局知识库的创建和管理
- [ ] 确保全局知识库对所有租户可见

### 11.6 Phase 6：知识图谱引擎集成（4-5周）

#### 引擎架构
- [ ] 实现 KnowledgeGraphEngine 接口
- [ ] 实现 GraphEngineManager
- [ ] 创建 WeKnora 引擎适配器
- [ ] 创建 GraphRAG 引擎适配器
- [ ] 创建 LightRAG 引擎适配器

#### 配置管理
- [ ] 设计图谱引擎配置结构
- [ ] 实现引擎配置的动态加载
- [ ] 实现引擎切换功能
- [ ] 添加引擎状态监控

#### 前端集成
- [ ] 实现图谱引擎选择界面
- [ ] 实现引擎配置管理页面
- [ ] 实现图谱重建进度显示
- [ ] 集成不同引擎的查询界面

### 11.7 Phase 7：测试与优化（2-3周）

#### 功能测试
- [ ] 编写权限系统集成测试
- [ ] 编写跨租户访问测试
- [ ] 编写分享功能端到端测试
- [ ] 编写知识图谱引擎切换测试

#### 性能测试
- [ ] 进行权限查询性能测试
- [ ] 进行大规模知识库列表查询测试
- [ ] 进行跨租户检索性能测试
- [ ] 优化查询性能和缓存策略

#### 安全测试
- [ ] 进行权限绕过安全测试
- [ ] 进行跨租户数据泄露测试
- [ ] 进行分享权限安全测试
- [ ] 完善安全审计机制

---

## 12. 总结

### 12.1 方案核心价值

本完整方案通过引入"所有权 + 可见性 + 分享"的三层权限模型，在保持现有租户隔离机制的基础上，成功实现了：

1. **全租户共享知识库**：系统级知识库，所有租户可访问
2. **租户私有知识库**：保持现有功能，向后兼容
3. **用户私有知识库**：个人知识管理，支持灵活分享
4. **知识库分享机制**：细粒度权限控制，支持过期和撤销

### 12.2 技术创新点

#### 12.2.1 资源租户概念
通过将 `TenantID` 语义升级为 `ResourceTenantID`，解决了跨租户访问时检索引擎选择的核心问题，确保共享知识库能够正确检索。

#### 12.2.2 多引擎知识图谱架构
设计了统一的知识图谱引擎接口，支持 WeKnora、GraphRAG、LightRAG 三种引擎的无缝切换，为不同场景提供最优解决方案。

#### 12.2.3 渐进式权限模型
从简单的租户隔离升级为多层次权限控制，既保证了安全性，又提供了灵活性。

### 12.3 实施保障

#### 12.3.1 向后兼容性
- 保持现有数据结构和API接口
- 通过功能开关控制新功能上线
- 提供完整的数据迁移方案

#### 12.3.2 可扩展性
- 模块化的权限服务设计
- 插件化的知识图谱引擎架构
- 灵活的配置管理机制

#### 12.3.3 可维护性
- 清晰的代码架构和接口设计
- 完善的测试覆盖
- 详细的文档和最佳实践

### 12.4 预期效果

通过本方案的实施，WeKnora 将从单一的租户隔离知识库系统，升级为支持多层次共享的企业级知识管理平台：

- **提升用户体验**：用户可以创建个人知识库，灵活分享知识
- **增强协作能力**：支持跨租户知识共享，促进组织间协作
- **扩展应用场景**：支持全局知识库，适用于更多业务场景
- **提高技术竞争力**：多引擎知识图谱支持，满足不同技术需求

### 12.5 风险控制

方案充分考虑了实施过程中的各种风险，并提供了相应的对策：

- **技术风险**：通过资源租户概念和引擎抽象解决
- **性能风险**：通过缓存和查询优化解决
- **安全风险**：通过完善的权限控制和审计机制解决
- **业务风险**：通过配额管理和监控机制解决

### 12.6 后续发展

本方案为 WeKnora 的后续发展奠定了坚实基础：

- **AI能力增强**：可基于知识图谱实现更智能的问答
- **多模态支持**：通过 LightRAG 等引擎支持图像、音频等多模态内容
- **企业级功能**：可扩展支持工作流、审批等企业级功能
- **生态集成**：可与更多第三方系统和服务集成

通过这个完整的改造方案，WeKnora 将成为一个真正意义上的现代化、企业级知识管理平台，能够满足不同规模组织的多样化需求。