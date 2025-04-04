#!/bin/bash

export CLUSTER_NAME=th-capa
export CLUSTERCLASS_NAME=aws-ec2-kubeadm-class
export NAMESPACE=default
export CONTROL_PLANE_MACHINE_COUNT=1
export AWS_REGION=us-west-2
export AWS_SSH_KEY_NAME=thehejik
export AWS_CONTROL_PLANE_MACHINE_TYPE=t3.medium
export AWS_NODE_MACHINE_TYPE=t3.medium
export KUBERNETES_VERSION=v1.31.0
export WORKER_MACHINE_COUNT=1

envsubst '${CLUSTER_NAME} ${CLUSTERCLASS_NAME} ${NAMESPACE} ${CONTROL_PLANE_MACHINE_COUNT} ${AWS_REGION} ${AWS_SSH_KEY_NAME} ${AWS_CONTROL_PLANE_MACHINE_TYPE} ${AWS_NODE_MACHINE_TYPE} ${KUBERNETES_VERSION} ${WORKER_MACHINE_COUNT}' < capa_kubeadm_upstream-classname.yaml > capa_kubeadm_upstream-classname-no-crs-rendered.yaml
