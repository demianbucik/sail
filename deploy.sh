#!/bin/bash

NAME=$1

gcloud alpha functions \
    deploy \
    "$NAME" \
    --gen2 \
    --trigger-http \
    --allow-unauthenticated \
    --source=. \
    --region=europe-west3 \
    --entry-point=SendEmailHandler \
    --memory=256MB \
    --cpu=1 \
    --runtime=go120 \
    --timeout=7 \
    --max-instances=2 \
    --concurrency=50 \
    --env-vars-file=./env.yaml
#    --docker-registry=artifact-registry \
