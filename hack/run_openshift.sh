# Delete old garbage
oc delete all --all

# We will be creating services, we need edit permissions
oc policy add-role-to-user edit system:serviceaccount:myproject:default -n `oc project` || true

# Ready, set, action
oc new-app jboss/infinispan-server:9.0.0.Final || true
oc scale dc infinispan-server --replicas 3