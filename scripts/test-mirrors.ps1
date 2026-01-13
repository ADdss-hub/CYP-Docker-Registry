# CYP-Docker-Registry Mirror Test Script
# Test connectivity and latency of Docker registry mirrors

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  CYP-Docker-Registry Mirror Speed Test" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$mirrors = @(
    @{ Name = "Docker Hub (Official)"; URL = "https://registry-1.docker.io/v2/" },
    @{ Name = "Aliyun Hangzhou"; URL = "https://registry.cn-hangzhou.aliyuncs.com/v2/" },
    @{ Name = "Aliyun Shanghai"; URL = "https://registry.cn-shanghai.aliyuncs.com/v2/" },
    @{ Name = "Aliyun Beijing"; URL = "https://registry.cn-beijing.aliyuncs.com/v2/" },
    @{ Name = "Tencent Cloud"; URL = "https://mirror.ccs.tencentyun.com/v2/" },
    @{ Name = "Huawei Cloud"; URL = "https://swr.cn-north-4.myhuaweicloud.com/v2/" },
    @{ Name = "Netease"; URL = "https://hub-mirror.c.163.com/v2/" },
    @{ Name = "USTC"; URL = "https://docker.mirrors.ustc.edu.cn/v2/" },
    @{ Name = "DaoCloud"; URL = "https://docker.m.daocloud.io/v2/" }
)

$results = @()

foreach ($mirror in $mirrors) {
    Write-Host "Testing: $($mirror.Name)" -ForegroundColor Yellow -NoNewline
    
    try {
        $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()
        $response = Invoke-WebRequest -Uri $mirror.URL -Method GET -TimeoutSec 10 -UseBasicParsing -ErrorAction Stop
        $stopwatch.Stop()
        
        $status = "OK"
        $latency = $stopwatch.ElapsedMilliseconds
        Write-Host " - OK ($latency ms)" -ForegroundColor Green
    }
    catch {
        $stopwatch.Stop()
        
        if ($_.Exception.Response.StatusCode.value__ -eq 401) {
            $status = "OK (Auth Required)"
            $latency = $stopwatch.ElapsedMilliseconds
            Write-Host " - OK (Auth Required, $latency ms)" -ForegroundColor Green
        }
        else {
            $status = "FAILED"
            $latency = 99999
            Write-Host " - FAILED" -ForegroundColor Red
        }
    }
    
    $results += [PSCustomObject]@{
        Name = $mirror.Name
        URL = $mirror.URL
        Status = $status
        Latency = $latency
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Results (sorted by latency)" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$available = $results | Where-Object { $_.Status -like "OK*" } | Sort-Object Latency

if ($available.Count -gt 0) {
    $rank = 1
    foreach ($item in $available) {
        Write-Host "  $rank. $($item.Name) - $($item.Latency) ms" -ForegroundColor White
        Write-Host "     $($item.URL)" -ForegroundColor Gray
        $rank++
    }
    
    Write-Host ""
    Write-Host "Recommended: $($available[0].Name)" -ForegroundColor Green
    Write-Host "URL: $($available[0].URL -replace '/v2/', '')" -ForegroundColor Green
}
else {
    Write-Host "No available mirrors!" -ForegroundColor Red
}

Write-Host ""
Write-Host "Done!" -ForegroundColor Cyan
