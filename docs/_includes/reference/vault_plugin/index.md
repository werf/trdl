## DESCRIPTION

The TRDL backend plugin allows publishing of project's releases into the TUF compatible repository.

## PATHS

The following paths are supported by this backend. To view help for
any of the paths below, use the help command with any route matching
the path pattern. Note that depending on the policy of your auth token,
you may or may not be able to access certain paths.

    ^configure/?$


    ^configure/git_credential/?$


    ^configure/pgp_signing_key$


    ^configure/trusted_pgp_public_key/(?P<name>\w(([\w-.]+)?\w)?)$


    ^configure/trusted_pgp_public_key/?$


    ^publish$
        Publish release channels

    ^release$
        Perform a release

    ^task/(?P<uuid>(?i:[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}))$


    ^task/(?P<uuid>(?i:[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}))/cancel$


    ^task/(?P<uuid>(?i:[0-9A-F]{8}-[0-9A-F]{4}-[4][0-9A-F]{3}-[89AB][0-9A-F]{3}-[0-9A-F]{12}))/log$


    ^task/?$


    ^task/configure/?$
