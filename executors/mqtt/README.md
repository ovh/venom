# Venom - Executor MQTT

Step to read and write MQTT topics.

An example can be found in `tests/mqtt.yml`

## Input

ClientType can be one of:
* "publisher"
* "subscriber"
* "persistent_queue"

As the name suggests "persistent_queue" creates a persistent queue by setting the session_clean property so that later "subscriber" steps can retrieve data. The "persistent_queue" is paired with "persistSubscription" which can be true or false, this will request/release the persistent topic subscription.
It is important that the persistent_queue and subscriber use the same client id to ensure the broker can track the state across connections. Remember to remove the topic registration when done.
Note the use of the name "persistent_queue" rather than MQTT's more usual clean_session. This is to reduce unexpected behaviour when one leaves the "persistent_queue" option out of the step config.

## Limitations and Future Improvements

### Limitations

Given the way that venom starts each task with no state held across steps and that mqtt would lose messages if it subscribed after a message is sent we need a mechanism to register persistent topics for a later mqtt step. This will require tests to add some extra steps to pre-register a persistent topic subscription and de-register at the end of a sequence of steps.

MQTT could deliver messages after connection, but before the service gets a chance to bind a subscription handler. For "subscriber" steps we attach a handler using AddRoute before connection and then subscribe after connection. This should ensure that no messages are lost and subscriptions behave as you need. This is needed to avoid message loss when a persistent subscription is being used.

### Future work 

* Add the ability to obtain message content from files rather than from within the yaml config. This need not be specific to this executor
* Add support for codecs so we can support serialisation formats other than json. This should really be a capability that any executor can take advantage of rather than solved for each individually
* Add TLs support for MQTT interface
