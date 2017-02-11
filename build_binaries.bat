@echo off

set os[0] = "windows"
set os[1] = "linux"
set arch[0] = "amd64"
set arch[1] = "386"

for %%o in (windows linux) do (
    for %%a in (amd64 386) do (
        setlocal
        set GOOS=%%o
        set GOARCH=%%a

        echo Running build for OS %%o using architecture %%a
        if %%o == windows (
            go build -o "bin\\POSbot_%%o_%%a.exe"
        ) else (
            go build -o "bin\\POSbot_%%o_%%a"
        )

        endlocal
    )
)