Add-Type -AssemblyName PresentationCore, PresentationFramework, WindowsBase

function Render-Badge {
    param(
        [string]$outputPath,
        [string]$svgPathData,
        [System.Windows.Media.Color]$bgColor,
        [System.Windows.Media.Color]$borderColor,
        [double]$targetSize = 300.0,
        [double]$borderThickness = 14.0
    )

    $width = 512
    $height = 512
    $dv = New-Object System.Windows.Media.DrawingVisual
    $dc = $dv.RenderOpen()

    # Draw circular background
    $brush = New-Object System.Windows.Media.SolidColorBrush ($bgColor)
    $borderBrush = New-Object System.Windows.Media.SolidColorBrush ($borderColor)
    $pen = New-Object System.Windows.Media.Pen ($borderBrush, $borderThickness)
    $dc.DrawEllipse($brush, $pen, (New-Object System.Windows.Point(256, 256)), 240, 240)

    # Parse and transform geometry
    $geom = [System.Windows.Media.Geometry]::Parse($svgPathData).Clone()
    $bounds = $geom.Bounds

    # Calculate scale factor to fit inside targetSize x targetSize
    $scaleX = $targetSize / $bounds.Width
    $scaleY = $targetSize / $bounds.Height
    $scale = [Math]::Min($scaleX, $scaleY)

    $scaledWidth = $bounds.Width * $scale
    $scaledHeight = $bounds.Height * $scale
    $offsetX = (512.0 - $scaledWidth) / 2.0 - ($bounds.X * $scale)
    $offsetY = (512.0 - $scaledHeight) / 2.0 - ($bounds.Y * $scale)

    $tg = New-Object System.Windows.Media.TransformGroup
    $tg.Children.Add((New-Object System.Windows.Media.ScaleTransform($scale, $scale)))
    $tg.Children.Add((New-Object System.Windows.Media.TranslateTransform($offsetX, $offsetY)))
    $geom.Transform = $tg

    $whiteBrush = New-Object System.Windows.Media.SolidColorBrush ([System.Windows.Media.Color]::FromArgb(255, 255, 255, 255))
    $dc.DrawGeometry($whiteBrush, $null, $geom)
    $dc.Close()

    $rtb = New-Object System.Windows.Media.Imaging.RenderTargetBitmap(512, 512, 96, 96, [System.Windows.Media.PixelFormats]::Pbgra32)
    $rtb.Render($dv)

    $enc = New-Object System.Windows.Media.Imaging.PngBitmapEncoder
    $enc.Frames.Add([System.Windows.Media.Imaging.BitmapFrame]::Create($rtb))
    
    $outDir = Split-Path -Parent $outputPath
    if (-not (Test-Path $outDir)) {
        New-Item -ItemType Directory -Path $outDir -Force | Out-Null
    }

    $stream = [System.IO.File]::Create($outputPath)
    $enc.Save($stream)
    $stream.Close()

    Write-Host "Generated $outputPath ($((Get-Item $outputPath).Length) bytes)"
}

$navyBg = [System.Windows.Media.Color]::FromArgb(255, 15, 24, 36)
$steelBorder = [System.Windows.Media.Color]::FromArgb(255, 74, 104, 138)

$crimsonBg = [System.Windows.Media.Color]::FromArgb(255, 42, 14, 20)
$crimsonBorder = [System.Windows.Media.Color]::FromArgb(255, 214, 48, 49)

# 1. Cruiser
$cruiserPath = "M-174-300h-6.989l4.989-9h2l5,4.5-5,4.5Zm-8.99,0H-187v-9h9l-4.989,9h0Z"
Render-Badge "assets/discord/cruiser.png" $cruiserPath $navyBg $steelBorder 320.0

# 2. Battleship
$bbPath = "M-174-287h-4.983l4.983-8.991V-296l5,4.5-5,4.5Zm-6.984,0h-2.005l4.988-9H-176l-4.988,9Zm-4.007,0H-187v-9h7l-4.989,9Z"
Render-Badge "assets/discord/battleship.png" $bbPath $navyBg $steelBorder 320.0

# 3. Destroyer
$ddPath = "M 0 0 L 17 4.5 L 0 9 Z"
Render-Badge "assets/discord/destroyer.png" $ddPath $navyBg $steelBorder 300.0

# 4. AirCarrier
$cvPath = "M 5 3 h 10 v 4 h -10 Z M 5 8 h 10 v 4 h -10 Z M 16 12 v -9 h 2 l 5 4.5 l -5 4.5 Z"
Render-Badge "assets/discord/aircarrier.png" $cvPath $navyBg $steelBorder 310.0

# 5. Submarine
$ssPath = "M-187-286.5h0v-9l1.734.459v8.082L-187-286.5Zm3.485-.922h0v-7.154L-170-291l-13.514,3.577Z"
Render-Badge "assets/discord/submarine.png" $ssPath $navyBg $steelBorder 300.0

# 6. Sunk (Sinking ship silhouette with waves)
$sunkPath = "M279.816 85.93L269.3 100.465l28.06 27.111 8.694-16.3zm-21.175 29.267l-42.282 58.436 37.246 35.984 34.946-65.525zM153.455 137.9l-46.816 29.192L306.46 360.15c4.08-1.929 8.605-3.15 13.539-3.15 3.881 0 7.508.76 10.867 2.02l-1.55-1.499 12.503-12.945L354.682 357l-10.268 10.63c4.31 3.924 8.02 8.35 11.486 12.745l11.077-11.465 12.859 12.426-12.508 12.945c3.62 4.436 8.724 9.265 13.916 10.43l10.908-11.293 5.016 4.846c3.702-3.574 7.58-8.457 11.63-13.664zm-98.623 4.168c-4.078 8.773-3.407 19.566-.8 32.7 3.406 17.167 10.435 36.207 10.222 54.75-.075 6.474-1.264 12.54-2.803 18.23l116.563 112.615C182.208 358.31 186.885 357 192 357c9.367 0 17.28 4.373 23.41 9.713 6.13 5.34 11.027 11.762 15.694 17.762 4.666 6 9.103 11.577 13.306 15.238 4.203 3.66 7.623 5.287 11.59 5.287 3.967 0 7.387-1.627 11.59-5.287s8.64-9.238 13.306-15.238c3.429-4.408 6.985-9.042 11.06-13.309zm106.947 28.569l12.862 12.425-12.508 12.946-12.862-12.426zm208.06 2.263l-10.517 14.536 28.059 27.109 8.693-16.299zM187.5 195.486l12.86 12.424-12.504 12.945-12.86-12.423zm161.165 6.682l-42.283 58.436 37.246 35.982 34.945-65.524zM213.22 220.334l12.861 12.424-12.504 12.945-12.861-12.424zm25.722 24.85l12.86 12.423-12.508 12.946-12.86-12.424zm221.084 15.191l-10.517 14.537 28.06 27.11 8.694-16.301zM55.56 267.084a66.9 66.9 0 0 0-.627 2.52c-2.532 11.232-2.865 18.802 7.882 29.185l89.479 86.453c.203-.26.4-.506.603-.767 3.338-4.292 6.802-8.794 10.743-12.967zm209.103 2.947l12.86 12.424-12.508 12.945-12.86-12.423zm174.188 19.612l-42.145 58.246 23.654 22.855a68.962 68.962 0 0 1 4.23-4.031C430.72 361.373 438.634 357 448 357c.082 0 .162.005.244.006l20.516-38.469zm-148.467 5.236l12.86 12.426-12.509 12.945-12.859-12.426zm25.72 24.85l12.86 12.423-12.508 12.946-12.86-12.424zM64 375c-19.807 5.72-33.274 18.147-46 31.898v30.579l.01-.006C31.525 428.62 47.7 419 64 419s27.881 10.654 37.842 19.154C111.802 446.654 120.144 453 128 453c7.856 0 16.197-6.346 26.158-14.846C164.12 429.654 175.7 419 192 419c16.3 0 27.881 10.654 37.842 19.154C239.802 446.654 248.144 453 256 453c7.856 0 16.197-6.346 26.158-14.846C292.12 429.654 303.7 419 320 419c16.3 0 27.881 10.654 37.842 19.154C367.802 446.654 376.144 453 384 453c7.856 0 16.197-6.346 26.158-14.846C420.12 429.654 431.7 419 448 419c16.3 0 32.475 9.62 45.99 18.47l.01.007v-30.578c-9.969-10.693-33.06-31.31-46-31.899-3.967 0-7.387 1.627-11.59 5.287C419.78 396.286 407.425 422.591 384 423c-27.151-1.916-36.8-28.488-52.41-42.713-4.203-3.66-7.623-5.287-11.59-5.287-3.967 0-7.387 1.627-11.59 5.287C291.78 396.286 279.425 422.591 256 423c-27.151-1.916-36.8-28.488-52.41-42.713-4.203-3.66-7.623-5.287-11.59-5.287C163.78 396.286 151.425 422.591 128 423c-27.151-1.916-36.8-28.488-52.41-42.713C71.387 376.627 67.967 375 64 375zm0 62c-18.195 3.791-32.267 12.364-46 22.385V494h476v-34.615c-11.752-8.054-32.671-21.938-46-22.385-24.766 6.02-41.304 33.45-64 34-23.532 2.01-46.008-36.297-64-34-25.282 3.227-41.304 33.45-64 34-23.532 2.01-46.008-36.297-64-34-25.282 3.227-41.304 33.45-64 34-23.532 2.01-45.87-33.455-64-34z"
Render-Badge "assets/discord/sunk.png" $sunkPath $crimsonBg $crimsonBorder 320.0
