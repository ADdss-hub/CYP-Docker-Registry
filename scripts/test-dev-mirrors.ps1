# Test Go and NPM mirrors for development dependencies

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Go Module Proxy Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$goProxies = @(
    @{ Name = "goproxy.cn (Qiniu)"; URL = "https://goproxy.cn" },
    @{ Name = "goproxy.io"; URL = "https://goproxy.io" },
    @{ Name = "mirrors.aliyun.com"; URL = "https://mirrors.aliyun.com/goproxy" },
    @{ Name = "proxy.golang.com.cn"; URL = "https://proxy.golang.com.cn" },
    @{ Name = "goproxy.baidu.com"; URL = "https://goproxy.baidu.com" },
    @{ Name = "Official (proxy.golang.org)"; URL = "https://proxy.golang.org" }
)

$goResults = @()

foreach ($proxy in $goProxies) {
    Write-Host "Testing: $($proxy.Name)" -ForegroundColor Yellow -NoNewline
    $testUrl = "$($proxy.URL)/github.com/gin-gonic/gin/@v/list"
    
    try {
        $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
        $response = Invoke-WebRequest -Uri $testUrl -Method GET -TimeoutSec 15 -UseBasicParsing -ErrorAction Stop
        $stopwatch.Stop()
        
        $latency = $stopwatch.ElapsedMilliseconds
        Write-Host " - OK ($latency ms)" -ForegroundColor Green
        $goResults += [PSCustomObject]@{ Name = $proxy.Name; URL = $proxy.URL; Latency = $latency; Status = "OK" }
    }
    catch {
        Write-Host " - FAILED" -ForegroundColor Red
        $goResults += [PSCustomObject]@{ Name = $proxy.Name; URL = $proxy.URL; Latency = 99999; Status = "FAILED" }
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  NPM Registry Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$npmRegistries = @(
    @{ Name = "npmmirror (Taobao)"; URL = "https://registry.npmmirror.com" },
    @{ Name = "Tencent"; URL = "https://mirrors.cloud.tencent.com/npm" },
    @{ Name = "Huawei"; URL = "https://repo.huaweicloud.com/repository/npm" },
    @{ Name = "cnpm"; URL = "https://r.cnpmjs.org" },
    @{ Name = "yarn China"; URL = "https://registry.npm.taobao.org" },
    @{ Name = "Official (npmjs.org)"; URL = "https://registry.npmjs.org" }
)

$npmResults = @()

foreach ($registry in $npmRegistries) {
    Write-Host "Testing: $($registry.Name)" -ForegroundColor Yellow -NoNewline
    $testUrl = "$($registry.URL)/vue"
    
    try {
        $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
        $response = Invoke-WebRequest -Uri $testUrl -Method GET -TimeoutSec 15 -UseBasicParsing -ErrorAction Stop
        $stopwatch.Stop()
        
        $latency = $stopwatch.ElapsedMilliseconds
        Write-Host " - OK ($latency ms)" -ForegroundColor Green
        $npmResults += [PSCustomObject]@{ Name = $registry.Name; URL = $registry.URL; Latency = $latency; Status = "OK" }
    }
    catch {
        Write-Host " - FAILED" -ForegroundColor Red
        $npmResults += [PSCustomObject]@{ Name = $registry.Name; URL = $registry.URL; Latency = 99999; Status = "FAILED" }
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Results Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

Write-Host ""
Write-Host "Go Proxy (sorted by latency):" -ForegroundColor Green
$goAvailable = $goResults | Where-Object { $_.Status -eq "OK" } | Sort-Object Latency
$rank = 1
foreach ($item in $goAvailable) {
    Write-Host "  $rank. $($item.Name) - $($item.Latency) ms" -ForegroundColor White
    $rank++
}

Write-Host ""
Write-Host "NPM Registry (sorted by latency):" -ForegroundColor Green
$npmAvailable = $npmResults | Where-Object { $_.Status -eq "OK" } | Sort-Object Latency
$rank = 1
foreach ($item in $npmAvailable) {
    Write-Host "  $rank. $($item.Name) - $($item.Latency) ms" -ForegroundColor White
    $rank++
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Recommended Configuration" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

if ($goAvailable.Count -gt 0) {
    $bestGo = $goAvailable[0]
    Write-Host "Go Proxy:" -ForegroundColor Yellow
    Write-Host "  go env -w GOPROXY=$($bestGo.URL),direct" -ForegroundColor White
    Write-Host "  go env -w GOSUMDB=sum.golang.google.cn" -ForegroundColor White
}

Write-Host ""

if ($npmAvailable.Count -gt 0) {
    $bestNpm = $npmAvailable[0]
    Write-Host "NPM Registry:" -ForegroundColor Yellow
    Write-Host "  npm config set registry $($bestNpm.URL)" -ForegroundColor White
}

Write-Host ""
Write-Host "Done!" -ForegroundColor Cyan
