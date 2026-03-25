# listicle shell integration for PowerShell (pwsh)
# Dot-source this file in your $PROFILE:
#   . /path/to/listicle/shell/listicle.ps1
#
# Or after `make install` it will be appended automatically.

function l {
    $tmp = [System.IO.Path]::GetTempFileName()
    listicle --cd-file $tmp @args
    $dir = Get-Content $tmp -ErrorAction SilentlyContinue
    Remove-Item $tmp -Force -ErrorAction SilentlyContinue
    if ($dir -and $dir -ne $PWD.Path) {
        Set-Location $dir
    }
}
