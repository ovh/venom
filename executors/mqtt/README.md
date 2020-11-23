# Venom - Executor MQTT

Step to use read / write on MQTT topics.

## Input

ClientType can be one of:
* "consumer"
* "producer"
* "persistent_queue"

"persistent_queue" prepares a queue by setting the session_clean property so that later "consumer" steps can retrieve data.
Note the use of the name "persistent_queue" rather than MQTT's more usual clean_session. This is as attempt to reduce unexpected behaviour when one leaves the "persistent_queue" option out of the step config.

## Limitations and Future Improvements

### Limitations

Given the way that venom starts each task with no state held across steps and that mqtt would lose messages if it subscribed after a message is sent we need a mechanism to register persistent topics for a later mqtt step. This will reduce our ability to test all combinations MQTT session settings. 

MQTT could deliver messages after connection, but before the service gets a chance to bind a subscription handler. For now we attach a global handler, however this could result in messages from unwanted topics creating false results in tests hence the reason for the TODO item to attach AddRoute calls to bind handlers only for chosen topics. This is undesired to have such a problem, so it's a high priority to resolve.

### Future work 

As mentioned in limitations above, we wish to limit subscriptions to only specified topics. 

The ability subscribe to multiple topics is likely to be important, especially if we wish to have a single persistent queue setup.

Add the ability to obtain message content from files rather than from within the yaml config.

It would be nice to support codecs so we can support serialisation formats other than json. This should really be a capability that any executor can take advantage of rather than solved for each individually.
