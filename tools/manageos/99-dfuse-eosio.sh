##
# This is place inside `/etc/profile.d/99-dfuse-eosio.sh`
# on built system an executed to provide message to use when they
# connect on the box.
export PATH=$PATH:/app

# If we are in a "node-manager" image, display special scripts motd
if [[ -d /nodeos-data/ ]]; then
    cat /etc/motd_node_manager
else
    cat /etc/motd_generic
fi
