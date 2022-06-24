#!/bin/bash
if [[ $1 = linux_* ]]; then
    upx --ultra-brute $2
fi
