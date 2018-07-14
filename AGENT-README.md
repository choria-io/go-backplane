## Choria Embeddable Backplane Client

This prepared the Choria RPC CLI and Ruby API for communication with services managed by the [Choria Embeddable Backplane](https://github.com/choria-io/go-backplane).

A compiled client with no dependencies can be found on the above page, if you only wish to interact with the backplane from the CLI that is the recommended tool to use.

<!--- actions -->

## Managing Backplanes

The backplane services are grouped together using subcollectives, for example if you have a service `notifications` those services will all inhabit the `notifications_backplane` sub collective.

To configure the Choria client to be able to communicate with these add this to your Hiera data:

```yaml
mcollective::client_collectives:
    - mcollective
    - notifications_backplane
```

You can then use any of the actions targeting the particular backplane, for example:

```
$ mco rpc backplane info -T notifications_backplane
```
