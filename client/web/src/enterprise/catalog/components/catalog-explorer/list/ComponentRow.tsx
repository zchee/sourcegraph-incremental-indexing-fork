import classNames from 'classnames'
import React from 'react'
import { Link } from 'react-router-dom'

import { isDefined } from '@sourcegraph/common'

import { Timestamp } from '../../../../../components/time/Timestamp'
import { ComponentListFields, ComponentRelationFields } from '../../../../../graphql-operations'
import { catalogRelationTypeDisplayName } from '../../../core/edges'
import { ComponentOwnerLink } from '../../component-owner-link/ComponentOwnerLink'
import { CatalogComponentIcon } from '../../ComponentIcon'

import styles from './CatalogExplorerList.module.scss'

export interface CatalogExplorerRowStyleProps {
    itemStartClassName?: string
    itemEndClassName?: string
    noBottomBorder?: boolean
}

interface Props extends CatalogExplorerRowStyleProps {
    node: ComponentListFields
    before?: string
}

export const ComponentRow: React.FunctionComponent<Props> = ({
    node,
    before,
    itemStartClassName,
    itemEndClassName,
    noBottomBorder,
}) => (
    <>
        {before && <span className={classNames('text-nowrap', itemStartClassName)}>{before}</span>}
        <h3 className={classNames('h6 font-weight-bold mb-0 d-flex align-items-center', !before && itemStartClassName)}>
            <Link to={node.catalogURL} className={classNames('d-block text-truncate')}>
                <CatalogComponentIcon
                    component={node}
                    className={classNames('icon-inline mr-1 flex-shrink-0 text-muted')}
                />
                {node.name}
            </Link>
        </h3>
        <ComponentOwnerLink owner={node.owner} className="text-nowrap" blankIfNone={true} />
        <span className="text-nowrap">{node.lifecycle?.toLowerCase()}</span>
        {node.__typename === 'Component' && node.commitsForLastCommitDate ? (
            <Timestamp
                className="text-nowrap"
                date={node.commitsForLastCommitDate.nodes[0].author.date}
                noAbout={true}
                strict={true}
            />
        ) : (
            <span />
        )}
        <div className={classNames('text-muted text-truncate', itemEndClassName)}>{node.description}</div>
        <div className={classNames({ 'border-top': !noBottomBorder }, styles.separator)} />
    </>
)

export const ComponentRowsHeader: React.FunctionComponent<
    Pick<CatalogExplorerRowStyleProps, 'itemStartClassName' | 'itemEndClassName'> & {
        before?: string
    }
> = ({ before, itemStartClassName, itemEndClassName }) => {
    const columns = [before, 'Name', 'Owner', 'Lifecycle', 'Last commit', 'Description'].filter(isDefined)
    return (
        <>
            {columns.map((text, index) => (
                <div
                    key={index}
                    className={classNames(
                        'text-muted mt-2 small',
                        index === 0 && itemStartClassName,
                        index === columns.length - 1 && itemEndClassName
                    )}
                >
                    {text}
                </div>
            ))}
            <div className={classNames('border-top', styles.separator)} />
        </>
    )
}

export const ComponentRelationRow: React.FunctionComponent<
    CatalogExplorerRowStyleProps & {
        edge: ComponentRelationFields
    }
> = ({ edge: { node, type }, ...props }) => (
    <>
        <ComponentRow {...props} node={node} before={catalogRelationTypeDisplayName(type)} />
    </>
)

export const ComponentRelationRowsHeader: React.FunctionComponent<
    Omit<React.ComponentPropsWithoutRef<typeof ComponentRowsHeader>, 'before'>
> = props => <ComponentRowsHeader {...props} before="Relation" />