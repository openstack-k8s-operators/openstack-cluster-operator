FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/openstack-cluster-operator \
    USER_UID=1001 \
    USER_NAME=openstack-cluster-operator \
    OPERATOR_BUNDLE=/usr/share/openstack-cluster-operator/bundle/

# install operator binary
COPY build/_output/bin/openstack-cluster-operator ${OPERATOR}
COPY build/_output/csv-merger /usr/local/bin/csv-merger

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# install CRDs and required roles, services, etc
RUN  mkdir -p ${OPERATOR_BUNDLE}
COPY deploy/crds/*crd.yaml ${OPERATOR_BUNDLE}

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
