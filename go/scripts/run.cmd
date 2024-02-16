@ECHO OFF

:: Bypass the "Terminate Batch Job" prompt
if "%~1"=="-FIXED_CTRL_C" (
   :: Remove the -FIXED_CTRL_C parameter
   SHIFT
) ELSE (
   :: Run the batch with <NUL and -FIXED_CTRL_C
   CALL <NUL %0 -FIXED_CTRL_C %*
   GOTO :EOF
)

SET PATH=%PATH%;C:\Windows\System32

set args=%1
shift
:start
if [%1] == [] goto done
set args=%args% %1
shift
goto start
:done

set /p version=< VERSION
set GORACE=history_size=7

nodemon --delay 1s -e go,html,yaml --signal SIGKILL --quiet ^
--exec "echo [Restarting] && go run -ldflags ""-X main.version=%VERSION%"" ./go/cmd/rpdb --config ./go/config.yaml" -- %args% || "exit 1"
