Sous update looks like this:

The DI injects a state manager, and the initial state read from that manager
  We build a HTTP state manager that does a GET /gdm to do that.

Update the overall state. Give the changed state to the state manager to write.
  (If the manifest doesn't exist yet...)
  The HSM receives this new state, and diffs with the state it had.
  It uses the diff concentrator to get diffing manifests
  For modifies: It GETs the pre-image manifests, confirms that they match the prior manifests in its diff,
    then PUTs with If-Match the post-manifests.
  For creates: It PUTs with If-None-Match: *
  For deletes: It GETs, compares and DELETEs with If-Match
  For retain: no action

  If there's errors (failed conditions), there's a balance between retrying
  inside the HSM or returning an error and letting the client retry.
  Ultimately, the HSM cannot tell what the client intended in terms of the change.
  `sous update` changes version fields in cluster config -
  if the update doesn't impact that,
  should we re-make the change and try again?
  Or report an error to the user to handle.

Which implies:
  Channels draining the manifest change channels, and triggering HTTP actions.

Next up: State != GDM; there's all the Defs component
  Which implies that the server needs to have an endpoint for the Defs
  (or for the State, but I lean towards keeping the State opaque...)
  (and actually, there's a REST design concern here: that's the appropriate model?)
  In the meantime GET /state-defs seems like an easy thing to build and helps with the HSM.
