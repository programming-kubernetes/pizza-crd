SHELL = bash

all: tls.key tls.crt pizza-crd.yaml serving-cert-secret.yaml

tls.key tls.crt:
		openssl req -new -x509 -subj "/CN=webhook.pizza-crd.svc" -nodes -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365

pizza-crd.yaml: tls.key tls.crt
		sed 's,CA_BUNDLE,$(shell base64 tls.crt),' pizza-crd.yaml.template > pizza-crd.yaml

serving-cert-secret.yaml: tls.key tls.crt
		sed 's,CERT,$(shell base64 tls.crt),;s,KEY,$(shell base64 tls.key),' serving-cert-secret.yaml.template > serving-cert-secret.yaml

clean:
		rm -f tls.key tls.crt pizza-crd.yaml serving-cert-secret.yaml