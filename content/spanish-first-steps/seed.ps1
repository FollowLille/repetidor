param(
    [long]$TrackId = 1,
    [string]$SqlitePath = "./repetidor.sqlite3",
    [switch]$Preview
)

$ErrorActionPreference = "Stop"
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "../..")
Push-Location $repoRoot

try {
    $files = @(
        "content/spanish-first-steps/00-spanish-first-steps.json",
        "content/spanish-first-steps/01-vocabulary-everyday.json",
        "content/spanish-first-steps/02-vocabulary-world.json",
        "content/spanish-first-steps/03-foundations.json",
        "content/spanish-first-steps/04-present-verbs.json",
        "content/spanish-first-steps/05-pronouns-daily.json",
        "content/spanish-first-steps/06-perfecto.json",
        "content/spanish-first-steps/07-indefinido.json"
    )

    foreach ($file in $files) {
        Write-Host "==> $file"
        $args = @(
            "run", "./cmd/course-seed",
            "-file", $file,
            "-sqlite-path", $SqlitePath,
            "-track-id", $TrackId
        )
        if ($Preview) {
            $args += "-preview"
        }
        & go @args
        if ($LASTEXITCODE -ne 0) {
            throw "course-seed failed for $file"
        }
    }
}
finally {
    Pop-Location
}
