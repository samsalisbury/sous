# API Change Guide

Sous transimits several of its data structures
via JSON
between client and server.
For the most part,
it should be possible to extend those structs
without causing problems,
so long as we follow a few simple rules.

Existing fields should not be changed.
Their types should remain the same,
and they certainly shouldn't be removed -
even if their values are no longer inspected.

New fields can be added reasonably freely,
with one caveat:
If the struct is used
as an item in a slice,
it's fields should be left as-is.
As a result,
it is probably preferable to add
maps of structs
rather than
lists of structs.

The reason for this
is that in order to
handle values that the client doesn't understand
(because, for instance,
the server has been updated since the client was installed)
they need to be compared,
and unchanged values returned unchanged.
A list can be compared,
but if any item in the list is changed,
the whole list has to be considered changed.
Consider: how would we determine which items were changed,
and which should be returned untouched.
So if there's a change anywhere in a list,
the whole list is replaced in the update.
Which means that if fields on the items were added,
they'll be destroyed on update.
(With a map, it's clear which items changed,
so we can isolate the change and retain fields we don't understand.)

## Canary Field

To protect against clients that do not yet know how to do these "safe" updates,
the server adds a field to its JSON responses.
The name of this field is unique to the request
(it's simply the Etag already generated).

The server expects to see this field
in PUTs with "If-Match" set,
(and extracts the name from the If-Match header)
and rejects updates without the field.

Properly behaved clients retain the field
the same way they retain any unrecognized field.
In fact, its purpose is effectively to be a field
that **no** client recognizes.

A client that discards the field
(because it doesn't match a field of its struct definitions)
signals that it might perform
inadvertantly destructive updates,
and so its updates cannot be trusted.

## Compatibility

This change was made after
version 0.5.14
so versions of Sous more recent than that should all be fine.
