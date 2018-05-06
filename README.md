# Minikube assign external IP

A service that listens for changes to kubernetes services, on finding a new service
that is of type load balancer it will assign it an external IP of the minikube
worker node.

With the ingress add-on enabled this sets a valid value for the external IP.
