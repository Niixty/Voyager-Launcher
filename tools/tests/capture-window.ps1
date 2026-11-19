param([int]$AppProcess,[string]$Destination)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.Drawing
Add-Type @'
using System;
using System.Runtime.InteropServices;
public class CaptureVoyager {
 [StructLayout(LayoutKind.Sequential)] public struct Rect { public int L,T,R,B; }
 [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hwnd,out Rect rect);
 [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr hwnd,IntPtr dc,uint flags);
 [DllImport("user32.dll")] public static extern bool SetProcessDPIAware();
}
'@
[void][CaptureVoyager]::SetProcessDPIAware()
$handle=(Get-Process -Id $AppProcess).MainWindowHandle
$rect=New-Object CaptureVoyager+Rect
if (-not [CaptureVoyager]::GetWindowRect($handle,[ref]$rect)) {throw 'Janela de teste ausente'}
$b=New-Object Drawing.Bitmap ($rect.R-$rect.L),($rect.B-$rect.T)
$graphics=[Drawing.Graphics]::FromImage($b)
$dc=$graphics.GetHdc()
try { if(-not [CaptureVoyager]::PrintWindow($handle,$dc,2)){throw 'Falha ao capturar janela'} } finally { $graphics.ReleaseHdc($dc) }
$b.Save([IO.Path]::GetFullPath($Destination),[Drawing.Imaging.ImageFormat]::Png)
$b.Dispose()
$graphics.Dispose()
