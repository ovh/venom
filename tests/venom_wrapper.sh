#!/bin/bash

DIR=`dirname $0`
NANO_TIMESTAMP=$(date +%s%N)

# Prepare args file from current args
rm -f $DIR/$NANO_TIMESTAMP.args.file
for v in "$@"
do
  echo $v >> $DIR/$NANO_TIMESTAMP.args.file
done

# Exec venom test binary, capture logs and exit code
$DIR/venom.test -test.coverprofile=$DIR/$NANO_TIMESTAMP.coverprofile -test.run="^TestBincoverRunMain$" -args-file=$DIR/$NANO_TIMESTAMP.args.file 1> $DIR/$NANO_TIMESTAMP.out 2> $DIR/$NANO_TIMESTAMP.error.out
EXI=$?

# Print venom log on stdout and go test log in a dedicated file
END_CTL_OUT=false
while IFS="" read -r p || [ -n "$p" ]
do
  if [[ $p == *"START_BINCOVER_METADATA" && "$p" != "START_BINCOVER_METADATA" ]]; then
    P=`printf '%s' "$p" | sed -e "s/START_BINCOVER_METADATA$//"`
    printf '%s\n' "$P"
    END_CTL_OUT=true
    echo "START_BINCOVER_METADATA" >> $DIR/$NANO_TIMESTAMP.test.out
  else
    if [ "$p" == "START_BINCOVER_METADATA" ]; then
      END_CTL_OUT=true
    fi
    if $END_CTL_OUT; then
      echo "$p" >> $DIR/$NANO_TIMESTAMP.test.out
    else
      printf '%s\n' "$p"
    fi
  fi
done < $DIR/$NANO_TIMESTAMP.out

echo "---- output ----"
cat $DIR/$NANO_TIMESTAMP.out

echo "---- error output ----"
cat $DIR/$NANO_TIMESTAMP.error.out

exit $EXI
