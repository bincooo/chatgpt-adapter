#!/usr/bin/env pwsh
<#
.SYNOPSIS
应用或还原 Go 模块的补丁文件。

.DESCRIPTION
遍历指定目录中的所有 .patch 文件，根据文件头 '# module: <module@version>' 找到对应的模块目录，
然后使用 git apply 应用或还原补丁。支持 vendor 目录优先。

.PARAMETER PatchDir
包含补丁文件的目录路径。

.PARAMETER Revert
开关，若指定则还原补丁（等效于 git apply -R）。

.PARAMETER Verbose
开关，输出更多详细信息。

.EXAMPLE
.\run.ps1 .\patches
应用 .\patches 目录下的所有补丁。

.EXAMPLE
.\run.ps1 -Revert .\patches
还原 .\patches 目录下的所有补丁。
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory=$true, Position=0)]
    [string]$PatchDir,

    [Parameter()]
    [switch]$Revert,

    [Parameter()]
    [switch]$Verbose
)

# 启用严格模式，类似 set -euo pipefail
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# ── 颜色定义 ─────────────────────────────────────────────────────────
# 使用 Write-Host 的颜色参数，定义一些便捷函数
function log   { Write-Host "[INFO] $args" -ForegroundColor Green }
function warn  { Write-Host "[WARN] $args" -ForegroundColor Yellow }
function error { Write-Host "[ERROR] $args" -ForegroundColor Red; exit 1 }

# ── 全局变量 ─────────────────────────────────────────────────────────
$PROJECT_ROOT = Get-Location
$GO_BIN = (Get-Command go -ErrorAction SilentlyContinue).Source
$GOMODCACHE = ""
$VERBOSE = $Verbose
$REVERT = $Revert

# ── 帮助信息（由 param 的注释处理，这里不再单独输出）───────────────

# ── 启动横幅 ──────────────────────────────────────────────────────────
function Print-Banner {
    Write-Host "==============================================" -ForegroundColor Blue
    Write-Host "         Go Module Patch Applicator           " -ForegroundColor Blue
    Write-Host "==============================================" -ForegroundColor Blue
}

# ── 环境检查 ──────────────────────────────────────────────────────────
function Check-Env {
    if (-not $GO_BIN) {
        error "未找到 go 命令"
    }
    if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
        error "未找到 git 命令（应用补丁需要）"
    }
    $script:GOMODCACHE = & go env GOMODCACHE
    if (-not $GOMODCACHE) {
        error "获取 GOMODCACHE 失败"
    }
}

# ── 从补丁头解析模块信息 ────────────────────────────────────────────────
function Parse-PatchModule {
    param([string]$PatchFile)
    # 读取第一行匹配 '# module:' 的内容
    $line = Select-String -Path $PatchFile -Pattern '^# module:' -List | ForEach-Object { $_.Line }
    if ($line) {
        # 去除 '# module:' 及周围空白
        $module = $line -replace '^# module:\s*', ''
        return $module.Trim()
    }
    return $null
}

# ── 提取模块路径（不含版本） ─────────────────────────────────────────────
function Module-Path {
    param([string]$Module)
    # 按 '@' 分割，取第一部分
    return ($Module -split '@')[0]
}

# ── 解析模块目录（优先 vendor，再使用 go list）───────────────────────────
function Resolve-ModuleDir {
    param([string]$Module)
    $path = Module-Path $Module
    $vendorDir = Join-Path $PROJECT_ROOT "vendor" $path
    if (Test-Path $vendorDir -PathType Container) {
        return $vendorDir
    }
    # 使用 go list 获取目录
    $dir = & go list -m -f '{{.Dir}}' $Module 2>$null
    if ($dir -and (Test-Path $dir -PathType Container)) {
        return $dir
    }
    return $null
}

# ── 安全应用或还原单个补丁 ───────────────────────────────────────────────
function Apply-Patch {
    param([string]$PatchFile)

    $action = if ($REVERT) { "还原" } else { "应用" }
    log "正在${action}补丁: $(Split-Path $PatchFile -Leaf)"

    $module = Parse-PatchModule $PatchFile
    if (-not $module) {
        warn "未找到模块头，跳过"
        return
    }

    if ($VERBOSE) { log "模块: $module" }

    $moduleDir = Resolve-ModuleDir $module
    if (-not $moduleDir) {
        warn "未找到模块目录: $module"
        return
    }

    if ($VERBOSE) { log "目录: $moduleDir" }

    # 确保有写权限（PowerShell 中无需 chmod，但可以确保目录可写）
    # 尝试进入目录
    Push-Location $moduleDir

    try {
        if ($REVERT) {
            # 还原模式：检查是否可以反向应用
            $check = & git apply --check -R $PatchFile 2>&1
            if ($LASTEXITCODE -eq 0) {
                & git apply -R $PatchFile
                if ($LASTEXITCODE -eq 0) {
                    log "补丁还原成功"
                } else {
                    warn "git apply -R 执行失败"
                }
            } else {
                warn "补丁还原跳过（可能尚未应用或冲突）"
                if ($VERBOSE) { Write-Host $check }
            }
        } else {
            # 应用模式
            $check = & git apply --check $PatchFile 2>&1
            if ($LASTEXITCODE -eq 0) {
                & git apply $PatchFile
                if ($LASTEXITCODE -eq 0) {
                    log "补丁应用成功"
                } else {
                    warn "git apply 执行失败"
                }
            } else {
                warn "补丁跳过（可能已应用或存在冲突）"
                if ($VERBOSE) { Write-Host $check }
            }
        }
    } finally {
        Pop-Location
    }
}

# ── 遍历所有补丁文件 ────────────────────────────────────────────────────
function Apply-AllPatches {
    $count = 0
    $patches = Get-ChildItem -Path $PatchDir -Filter "*.patch" -File
    foreach ($patch in $patches) {
        Apply-Patch $patch.FullName
        $count++
        Write-Host
    }

    if ($REVERT) {
        log "总共还原的补丁数: $count"
    } else {
        log "总共处理的补丁数: $count"
    }
}

# ── 主函数 ────────────────────────────────────────────────────────────
function Main {
    # 验证补丁目录
    if (-not (Test-Path $PatchDir -PathType Container)) {
        error "补丁目录不存在: $PatchDir"
    }

    Print-Banner
    Check-Env

    log "补丁目录: $PatchDir"
    log "Go 模块缓存: $GOMODCACHE"
    Write-Host

    Apply-AllPatches
    log "完成。"
}

# 执行主函数
Main