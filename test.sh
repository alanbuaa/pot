#!/bin/bash

startTime=`date +%Y%m%d-%H:%M:%S`
startTime_s=`date +%s`

/work/crypto/vdf/utils/vdf-linux  c26f92445addcd36b5d58b681f90a9bda23af1f611a0b8f53b99b39afba9df95 300000

endTime=`date +%Y%m%d-%H:%M:%S`
endTime_s=`date +%s`

sumTime=$[ $endTime_s - $startTime_s ]

echo "$startTime ---> $endTime" "Total:$sumTime seconds"