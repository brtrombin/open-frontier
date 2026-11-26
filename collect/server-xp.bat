@echo off
rem SPDX-License-Identifier: GPL-2.0-only
rem
rem open-frontier read-only collection kit: FRONT-END SERVER (Windows XP, also works on 7/10).
rem
rem What it does: runs built-in Windows listing commands and writes their
rem output to a new folder on the USB stick this file runs from.
rem What it never does: talk to the scanner, change settings, delete files,
rem write to the registry, or copy programs/drivers (.exe .dll .sys).
rem
rem Run it from the USB stick (double click) when no scan is running.
rem IMPORTANT: running this while a scan is in progress can cause a sharing
rem violation that aborts ExecLSeqObj.exe and wedges the scanner.

setlocal
set DRV=%~d0

rem Refuse to run from the system disk: output must go to the USB stick.
if /i "%DRV%"=="C:" goto wrongdrive

set N=1
:findname
set OUT=%DRV%\collect-%COMPUTERNAME%-%N%
if exist "%OUT%\README.txt" (
  set /a N=N+1
  goto findname
)
mkdir "%OUT%"
mkdir "%OUT%\registry"
mkdir "%OUT%\OPTIONAL-may-contain-licenses"
mkdir "%OUT%\config-files"

echo.
echo open-frontier: coleta somente leitura / read-only collection
echo Saida / output: %OUT%
echo Nao feche esta janela / do not close this window.
echo.

echo [1/9] Sistema / system...
(
  echo open-frontier collection kit, front-end server
  echo Read-only. Contains machine configuration. Share privately, never publish.
  date /t
  time /t
  ver
  echo COMPUTERNAME=%COMPUTERNAME%
  echo USERNAME=%USERNAME%
) > "%OUT%\README.txt"
systeminfo > "%OUT%\system.txt" 2>&1
set >> "%OUT%\system.txt" 2>&1

echo [2/9] Rede / network...
ipconfig /all > "%OUT%\network.txt" 2>&1
net share >> "%OUT%\network.txt" 2>&1
net use >> "%OUT%\network.txt" 2>&1
net config workstation >> "%OUT%\network.txt" 2>&1
net config server >> "%OUT%\network.txt" 2>&1

echo [3/9] Servicos e drivers / services and drivers...
net start > "%OUT%\services-running.txt" 2>&1
sc query type= service state= all > "%OUT%\services-all.txt" 2>&1
driverquery /v > "%OUT%\drivers.txt" 2>&1
dir /a "%SystemRoot%\system32\drivers" > "%OUT%\drivers-list.txt" 2>&1

echo [4/9] Registro / registry (hardware)...
reg export "HKLM\SYSTEM\CurrentControlSet\Enum" "%OUT%\registry\enum.reg" > nul 2>&1
reg export "HKLM\SYSTEM\CurrentControlSet\Services" "%OUT%\registry\services.reg" > nul 2>&1
reg export "HKLM\SYSTEM\CurrentControlSet\Control\Class" "%OUT%\registry\class.reg" > nul 2>&1

echo [5/9] Registro / registry (Fujifilm)...
for %%K in ("HKLM\SOFTWARE\FUJIFILM" "HKLM\SOFTWARE\Fuji Photo Film" "HKLM\SOFTWARE\FUJI PHOTO FILM CO.,LTD." "HKLM\SOFTWARE\Frontier") do (
  reg query %%K /s >> "%OUT%\registry\fujifilm-keys.txt" 2>&1
)

echo [6/9] Registro / registry (software, opcional)...
reg export "HKLM\SOFTWARE" "%OUT%\OPTIONAL-may-contain-licenses\software.reg" > nul 2>&1

echo [7/10] InSpool e ImgExchange (lista / listing only)...
if exist "D:\InSpool" dir /a "D:\InSpool" > "%OUT%\inspool-listing.txt" 2>&1
if exist "D:\OutSpool" dir /a "D:\OutSpool" > "%OUT%\outspool-listing.txt" 2>&1
if exist "D:\ImgExchange" dir /a "D:\ImgExchange" > "%OUT%\imgexchange-listing.txt" 2>&1

echo [8/10] Lista de arquivos / file listing (pode demorar / may take a while)...
dir /s /a C:\ > "%OUT%\files-C.txt" 2>&1
for %%D in (D E F) do (
  if /i not "%%D:"=="%DRV%" if exist %%D:\ dir /s /a %%D:\ > "%OUT%\files-%%D.txt" 2>&1
)

echo [9/10] Arquivos de configuracao / config files (.ini .cfg)...
for /d %%F in ("C:\Fuji*" "D:\Fuji*" "C:\Frontier*" "D:\Frontier*" "C:\Program Files\Fuji*" "C:\Program Files\Frontier*" "C:\Program Files\Sprocket*") do (
  xcopy "%%F\*.ini" "%OUT%\config-files\%%~nxF\" /s /i /c /h /y /q >> "%OUT%\config-files\xcopy.log" 2>&1
  xcopy "%%F\*.cfg" "%OUT%\config-files\%%~nxF\" /s /i /c /h /y /q >> "%OUT%\config-files\xcopy.log" 2>&1
)

echo [10/10] Relatorio de hardware / hardware report (msinfo32, alguns minutos / a few minutes)...
if exist "%CommonProgramFiles%\Microsoft Shared\MSInfo\msinfo32.exe" (
  start /wait "" "%CommonProgramFiles%\Microsoft Shared\MSInfo\msinfo32.exe" /report "%OUT%\msinfo32.txt"
) else (
  echo msinfo32.exe not found > "%OUT%\msinfo32.txt"
)

echo. >> "%OUT%\README.txt"
echo Finished: >> "%OUT%\README.txt"
date /t >> "%OUT%\README.txt"
time /t >> "%OUT%\README.txt"

echo.
echo PRONTO / DONE: %OUT%
echo Pode fechar a janela / you can close this window.
pause
goto end

:wrongdrive
echo.
echo ERRO: rode este arquivo a partir do PENDRIVE, nao de %DRV%
echo ERROR: run this file from the USB stick, not from %DRV%
pause

:end
endlocal
