#!/bin/bash

scp -r * root@172.17.80.1:/home/cv/ 

#rsync  -avzhe ssh  --filter "- *.json" --filter "- *.zip" --filter "- *.yaml" ./ root@172.17.80.1:/home/cv/ 