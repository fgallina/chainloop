FROM quay.io/keycloak/keycloak:23.0.4 as builder

WORKDIR /opt/keycloak
RUN /opt/keycloak/bin/kc.sh build --db=postgres

FROM quay.io/keycloak/keycloak:23.0.4
COPY --from=builder /opt/keycloak/ /opt/keycloak/

ENTRYPOINT ["/opt/keycloak/bin/kc.sh"]
