
# Delete old garbage
oc delete all --all

# We will be creating services, we need edit permissions
oc policy add-role-to-user edit system:serviceaccount:myproject:default -n myproject || true

# Ready, set, action
oc new-app slaskawi/external-ip-proxy || true