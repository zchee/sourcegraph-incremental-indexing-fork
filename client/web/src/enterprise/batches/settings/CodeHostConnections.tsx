import React from 'react'

import { Container, Link } from '@sourcegraph/wildcard'

import { UseConnectionResult } from '../../../components/FilteredConnection/hooks/useConnection'
import {
    ConnectionContainer,
    ConnectionError,
    ConnectionList,
    ConnectionLoading,
    ConnectionSummary,
    ShowMoreButton,
    SummaryContainer,
} from '../../../components/FilteredConnection/ui'
import { BatchChangesCodeHostFields, Scalars } from '../../../graphql-operations'

import { useGlobalBatchChangesCodeHostConnection, useUserBatchChangesCodeHostConnection } from './backend'
import { CodeHostConnectionNode } from './CodeHostConnectionNode'

export interface GlobalCodeHostConnectionsProps {
    headerLine: JSX.Element
}

export const GlobalCodeHostConnections: React.FunctionComponent<GlobalCodeHostConnectionsProps> = props => (
    <CodeHostConnections userID={null} connectionResult={useGlobalBatchChangesCodeHostConnection()} {...props} />
)

export interface UserCodeHostConnectionsProps extends GlobalCodeHostConnectionsProps {
    userID: Scalars['ID']
}

export const UserCodeHostConnections: React.FunctionComponent<UserCodeHostConnectionsProps> = props => (
    <CodeHostConnections connectionResult={useUserBatchChangesCodeHostConnection(props.userID)} {...props} />
)

interface CodeHostConnectionsProps extends GlobalCodeHostConnectionsProps {
    userID: Scalars['ID'] | null
    connectionResult: UseConnectionResult<BatchChangesCodeHostFields>
}

const CodeHostConnections: React.FunctionComponent<CodeHostConnectionsProps> = ({
    userID,
    headerLine,
    connectionResult,
}) => {
    const { loading, hasNextPage, fetchMore, connection, error, refetchAll } = connectionResult
    return (
        <Container>
            <h3>Code host tokens</h3>
            {headerLine}
            <ConnectionContainer className="mb-3">
                {error && <ConnectionError errors={[error.message]} />}
                {loading && !connection && <ConnectionLoading />}
                <ConnectionList as="ul" className="list-group">
                    {connection?.nodes?.map(node => (
                        <CodeHostConnectionNode
                            key={node.externalServiceURL}
                            node={node}
                            refetchAll={refetchAll}
                            userID={userID}
                        />
                    ))}
                </ConnectionList>
                {connection && (
                    <SummaryContainer className="mt-2">
                        <ConnectionSummary
                            noSummaryIfAllNodesVisible={true}
                            first={15}
                            connection={connection}
                            noun="code host"
                            pluralNoun="code hosts"
                            hasNextPage={hasNextPage}
                        />
                        {hasNextPage && <ShowMoreButton onClick={fetchMore} />}
                    </SummaryContainer>
                )}
            </ConnectionContainer>
            <p className="mb-0">
                Code host not present? Site admins can add a code host in{' '}
                <Link to="/help/admin/external_service" target="_blank" rel="noopener noreferrer">
                    the manage repositories settings
                </Link>
                .
            </p>
        </Container>
    )
}
