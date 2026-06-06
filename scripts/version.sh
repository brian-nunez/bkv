#!/bin/bash

TAG=$1

# Tag with path-relative names that match module import path
git tag $TAG
git tag drivers/local/$TAG
git tag drivers/redis/$TAG
git tag drivers/sqlite/$TAG

# Push the correct tags
git push origin $TAG
git push origin drivers/local/$TAG
git push origin drivers/redis/$TAG
git push origin drivers/sqlite/$TAG
