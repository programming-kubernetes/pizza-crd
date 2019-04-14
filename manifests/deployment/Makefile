SHELL = bash

all: tls.key tls.crt pizza-crd.yaml

tls.key tls.crt:
		openssl req -new -x509 -subj "/CN=webhook.pizza.drc.svc.cluster.local" -nodes -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365

pizza-crd.yaml: tls.key tls.crt
		sed 's,CA_BUNDLE,$(shell base64 tls.crt),' pizza-crd.yaml.template > pizza-crd.yaml

clean:
		rm -f tls.key tls.crt pizza-crd.yaml