#!/usr/bin/env bash

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ]; do # resolve $SOURCE until the file is no longer a symlink
  DIR="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
  SOURCE="$(readlink "$SOURCE")"
  [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE" # if $SOURCE was a relative symlink, we need to resolve it relative to the path where the symlink file was located
done
HERE="$( cd -P "$( dirname "$SOURCE" )" && pwd )"
echo ${HERE}

systemctl stop eso-trials-bot
cp ${HERE}/eso-trials-bot.service /etc/systemd/system/
systemctl daemon-reload
rm ${HERE}/trials-bot
gunzip ${HERE}/trials-bot.gz
systemctl start eso-trials-bot
rm ${HERE}/trials-dump
gunzip ${HERE}/trials-dump