# Operator

A skeleton k8s operator project.

## Instructions

Note:

        operator/ is the directory containing this cloned repository
        workspace/operator/ is the directory where the operator is developed

1. Build and install the Operator SDK

        $ git clone https://github.com/operator-framework/operator-sdk
        $ cd operator-sdk
        $ make install

2. Deploy the operator web app container

        $ cd operator
        $ docker build -t localhost:5000/operatorwebapp:latest operatorwebapp
        $ docker push localhost:5000/operatorwebapp:latest
        $ curl http://localhost:5000/v2/operatorwebapp/tags/list

3. Create a workspace

        Note: the name of the directory 'operator' is used by the SDK to name pods, etc.

        $ mkdir -p workspace/operator
        $ cd workspace/operator

4. Create a sample operator

        $ export GO111MODULE=on
        $ operator-sdk init --domain example.org --repo github.com/keith-cullen/operator

        edit Makefile

            VERSION ?= latest
            IMAGE_TAG_BASE ?= localhost:5000/operator
            IMG ?= $(IMAGE_TAG_BASE):$(VERSION)

5. Create an API and controller

        $ operator-sdk create api --group cache --version v1alpha1 --kind Operator --resource --controller
        $ cp ../../operator_types.go api/v1alpha1/operator_types.go
        $ cp ../../operator_controller.go controllers/operator_controller.go
        $ make generate
        $ make manifests

6. Build the operator

        Note: each time the operator is built, the value of the VERSION variable in the Makefile must be changed from that of the previous build

        $ make docker-build docker-push

7. (Optionally) Run the Operator locally

        $ make install run

8. Deploy the operator

        $ make deploy

9. Deploy a custom resource

        $ cp ../../cache_v1alpha1_operator.yaml .
        $ kubectl apply -f cache_v1alpha1_operator.yaml
        $ kubectl get deployment -A
        $ kubectl get pods -A
        $ kubectl get operator/operator -o yaml

10. Interact with deployment

        $ kubectl get pods -A -owide

        Note: the operator IP addresses

        $ curl http://<IP>:8081

11. Clean-up

        $ kubectl delete -f cache_v1alpha1_operator.yaml
        $ make undeploy
