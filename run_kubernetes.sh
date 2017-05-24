# Ready, set, action
oc new-app jboss/infinispan-server:9.0.0.Final || true
oc scale dc infinispan-server --replicas 3