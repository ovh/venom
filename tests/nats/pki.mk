$(NATS_PKI_DIR):
	mkdir -p $(NATS_PKI_DIR)

	# create a certificate authority (CA) that both the client and server trust.
	# The CA is just a public and private key with the public key wrapped up in a self-signed X.509 certificate.
	openssl req \
		-new \
		-x509 \
		-nodes \
		-days 365 \
		-subj '/C=GB/O=Example/OU=TeamA/CN=ca.example.com' \
		-keyout $(NATS_PKI_DIR)/ca.key \
		-out $(NATS_PKI_DIR)/ca.crt

	# create the server’s key
	openssl genrsa \
		-out $(NATS_PKI_DIR)/server.key 2048

	# create a server Certificate Signing Request
	openssl req \
		-new \
		-key $(NATS_PKI_DIR)/server.key \
		-subj '/C=GB/O=Example/OU=TeamA/CN=example.com' \
		-addext 'subjectAltName = DNS:venom-$(nats-server-name), DNS:localhost, IP:127.0.0.1, IP:::1' \
		-out $(NATS_PKI_DIR)/server.csr

	# creates the server signed certificate
	openssl x509 \
		-req \
		-in $(NATS_PKI_DIR)/server.csr \
		-CA $(NATS_PKI_DIR)/ca.crt \
		-CAkey $(NATS_PKI_DIR)/ca.key \
		-CAcreateserial \
		-days 365 \
		-copy_extensions copy \
		-out $(NATS_PKI_DIR)/server.crt

	# create the client's key
	openssl genrsa \
		-out $(NATS_PKI_DIR)/client.key 2048

	# create a client Certificate Signing Request
	openssl req \
		-new \
		-key $(NATS_PKI_DIR)/client.key \
		-subj '/CN=user1.example.com' \
		-out $(NATS_PKI_DIR)/client.csr

	# creates the client signed certificate
	openssl x509 \
		-req \
		-in $(NATS_PKI_DIR)/client.csr \
		-CA $(NATS_PKI_DIR)/ca.crt \
		-CAkey $(NATS_PKI_DIR)/ca.key \
		-CAcreateserial \
		-days 365 \
		-out $(NATS_PKI_DIR)/client.crt

.PHONY: clean-nats-pki
clean-nats-pki:
	rm -fr $(NATS_PKI_DIR)