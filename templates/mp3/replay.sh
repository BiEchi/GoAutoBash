#!/bin/bash

YELLOW='\033[1;33m'
NC='\033[0m'  # No Color

if [ "$#" -ne 1 ] || ! [ -z "${1//[0-9]}" ]; then
  echo "Usage: $0 index (decimal number)" >&2
  exit 1
fi

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd "${DIR}"

basename="test$(printf %06d $1)"

if ! [ -d ${basename} ]; then
  if ! [ -d "${basename}-regression" ]; then
    echo "Fail to find test case ${basename} or ${basename}-regression. Make sure the index is valid."
    exit 1
  else
    basename="${basename}-regression"
  fi
fi

for filename in *.asm; do
    if ! lc3as "$filename" >/dev/null; then
      echo "Fail to compile test data ${filename}. Please contact TAs on Piazza."
      exit 1
    fi
done

for filename in ${basename}/*.asm; do
    if ! lc3as "$filename" >/dev/null; then
      echo "Fail to compile test data ${filename}. Please contact TAs on Piazza."
      exit 1
    fi
done

if ! command -v expect &> /dev/null
then
    echo -e "[NOTE] Type in \"${YELLOW}execute ${basename}/${basename}.lcs${NC}\" in lc3sim to load the test case."
    echo -e "[NOTE] If you are not working on a public machine, you may install expect to automate this process."
    echo -e "       Use \"sudo apt-get install -y expect\" to install expect."
    lc3sim
else
    expect -c "spawn lc3sim; expect \"(lc3sim) \"; send \"execute ${basename}/${basename}.lcs\n\"; interact"
fi