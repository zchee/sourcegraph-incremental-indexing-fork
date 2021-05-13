import classNames from 'classnames'
import * as React from 'react'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { LinkOrSpan } from '@sourcegraph/shared/src/components/LinkOrSpan'
import { gql } from '@sourcegraph/shared/src/graphql/graphql'
import { createAggregateError } from '@sourcegraph/shared/src/util/errors'
import { memoizeObservable } from '@sourcegraph/shared/src/util/memoizeObservable'
import { numberWithCommas } from '@sourcegraph/shared/src/util/strings'

import { requestGraphQL } from '../backend/graphql'
import { Timestamp } from '../components/time/Timestamp'
import {
    GitRefConnectionFields,
    GitRefFields,
    GitRefType,
    RepositoryGitRefsResult,
    RepositoryGitRefsVariables,
    Scalars,
} from '../graphql-operations'

interface GitReferenceNodeProps {
    node: GitRefFields

    /** Link URL; if undefined, node.url is used. */
    url?: string

    /** Whether any ancestor element higher up in the tree is an `<a>` element. */
    ancestorIsLink?: boolean

    children?: React.ReactNode

    className?: string
}

export const GitReferenceNode: React.FunctionComponent<GitReferenceNodeProps> = ({
    node,
    url,
    ancestorIsLink,
    children,
    className,
}) => {
    const mostRecentSig =
        node.target.commit &&
        (node.target.commit.committer && node.target.commit.committer.date > node.target.commit.author.date
            ? node.target.commit.committer
            : node.target.commit.author)
    const behindAhead = node.target.commit?.behindAhead
    url = url !== undefined ? url : node.url

    return (
        <LinkOrSpan
            key={node.id}
            className={classNames('git-ref-node list-group-item', className)}
            to={!ancestorIsLink ? url : undefined}
        >
            <span>
                <code className="git-ref-tag-2">{node.displayName}</code>
                {mostRecentSig && (
                    <small className="pl-2">
                        Updated <Timestamp date={mostRecentSig.date} />{' '}
                        {mostRecentSig.person && <>by {mostRecentSig.person.displayName}</>}
                    </small>
                )}
            </span>
            {behindAhead && (
                <small>
                    {numberWithCommas(behindAhead.behind)} behind, {numberWithCommas(behindAhead.ahead)} ahead
                </small>
            )}
            {children}
        </LinkOrSpan>
    )
}

export const gitReferenceFragments = gql`
    fragment GitRefFields on GitRef {
        id
        displayName
        name
        abbrevName
        url
        target {
            commit {
                author {
                    ...SignatureFieldsForReferences
                }
                committer {
                    ...SignatureFieldsForReferences
                }
                behindAhead(revspec: "HEAD") @include(if: $withBehindAhead) {
                    behind
                    ahead
                }
            }
        }
    }

    fragment SignatureFieldsForReferences on Signature {
        person {
            displayName
            user {
                username
            }
        }
        date
    }
`

export const queryGitReferences = memoizeObservable(
    (args: {
        repo: Scalars['ID']
        first?: number
        query?: string
        type: GitRefType
        withBehindAhead?: boolean
    }): Observable<GitRefConnectionFields> =>
        requestGraphQL<RepositoryGitRefsResult, RepositoryGitRefsVariables>(
            gql`
                query RepositoryGitRefs(
                    $repo: ID!
                    $first: Int
                    $query: String
                    $type: GitRefType!
                    $withBehindAhead: Boolean!
                ) {
                    node(id: $repo) {
                        ... on Repository {
                            gitRefs(first: $first, query: $query, type: $type, orderBy: AUTHORED_OR_COMMITTED_AT) {
                                ...GitRefConnectionFields
                            }
                        }
                    }
                }

                fragment GitRefConnectionFields on GitRefConnection {
                    nodes {
                        ...GitRefFields
                    }
                    totalCount
                    pageInfo {
                        hasNextPage
                    }
                }

                ${gitReferenceFragments}
            `,
            {
                query: args.query ?? null,
                first: args.first ?? null,
                repo: args.repo,
                type: args.type,
                withBehindAhead:
                    args.withBehindAhead !== undefined ? args.withBehindAhead : args.type === GitRefType.GIT_BRANCH,
            }
        ).pipe(
            map(({ data, errors }) => {
                if (!data || !data.node || !data.node.gitRefs) {
                    throw createAggregateError(errors)
                }
                return data.node.gitRefs
            })
        ),
    args => `${args.repo}:${String(args.first)}:${String(args.query)}:${args.type}`
)
