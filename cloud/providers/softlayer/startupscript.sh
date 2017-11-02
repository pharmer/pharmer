#!/bin/bash

# ref: https://gist.github.com/przemoc/571091
# ref: https://stackoverflow.com/a/1985512/244009

LOCKFILE="/var/lock/pharmer"
LOCKFD=99

# PRIVATE
_lock()             { flock -$1 $LOCKFD; }
_no_more_locking()  { _lock u; _lock xn && rm -f $LOCKFILE; }
_prepare_locking()  { eval "exec $LOCKFD>\"$LOCKFILE\""; trap _no_more_locking EXIT; }

# ON START
_prepare_locking

# PUBLIC
exlock_now()        { _lock xn; }  # obtain an exclusive lock immediately or fail
exlock()            { _lock x; }   # obtain an exclusive lock
shlock()            { _lock s; }   # obtain a shared lock
unlock()            { _lock u; }   # drop a lock

### BEGIN OF SCRIPT ###

# Simplest example is avoiding running multiple instances of script.
exlock_now || exit 1

/usr/bin/wget -O /usr/bin/pharmer.sh https://api.service.softlayer.com/rest/v3/SoftLayer_Resource_Metadata/UserMetadata.txt
chmod +x /usr/bin/pharmer.sh
/usr/bin/pharmer.sh

# Remember! Lock file is removed when one of the scripts exits and it is
#           the only script holding the lock or lock is not acquired at all.
