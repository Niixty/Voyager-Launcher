# Select a file in this fixture's native dialog without keyboard focus or mouse coordinates.
param([int]$AppProcess,[string]$ImagePath)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes
$root=[System.Windows.Automation.AutomationElement]::RootElement
$scope=[System.Windows.Automation.TreeScope]
$ae=[System.Windows.Automation.AutomationElement]
function Find-Property($parent,$prop,$value) {
    $test=[System.Windows.Automation.PropertyCondition]::new($prop,$value)
    return $parent.FindFirst($scope::Descendants,$test)
}
$test=[System.Windows.Automation.PropertyCondition]::new($ae::ProcessIdProperty,$AppProcess)
$end=(Get-Date).AddSeconds(20)
while ((Get-Date) -lt $end) {
    foreach($window in $root.FindAll($scope::Children,$test)) {
        if ($window.Current.Name -eq 'Selecionar arquivo') { $dialog=$window;break }
        $dialog=Find-Property $window $ae::NameProperty 'Selecionar arquivo'
        if ($dialog) {break}
    }
    if($dialog) {break}
    Start-Sleep -Milliseconds 200
}
if(-not $dialog){throw 'Seletor nativo nao abriu'}
$edit=Find-Property $dialog $ae::ControlTypeProperty ([System.Windows.Automation.ControlType]::Edit)
$accept=Find-Property $dialog $ae::AutomationIdProperty '1'
if (-not $edit -or -not $accept) {throw 'Controles do seletor ausentes'}
$edit.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern).SetValue($ImagePath)
$accept.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern).Invoke()
