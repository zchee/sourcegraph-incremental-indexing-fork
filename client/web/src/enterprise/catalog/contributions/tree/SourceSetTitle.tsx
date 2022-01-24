import FolderIcon from 'mdi-react/FolderIcon'
import React from 'react'

import { ComponentKind } from '@sourcegraph/shared/src/schema'

import { CatalogComponentIcon } from '../../components/ComponentIcon'

import { SourceSetAtTreeViewOptionsProps } from './useSourceSetAtTreeViewOptions'

interface Props extends Pick<SourceSetAtTreeViewOptionsProps, 'sourceSetAtTreeViewMode'> {
    component: {
        __typename: 'Component'
        name: string
        kind: ComponentKind
    } | null
    tree: {
        path: string
    }
}

export const SourceSetTitle: React.FunctionComponent<Props> = ({ component, tree, sourceSetAtTreeViewMode }) =>
    sourceSetAtTreeViewMode === 'auto' && component ? (
        <ComponentTitleWithIconAndKind component={component} />
    ) : (
        <>
            <FolderIcon className="icon-inline mr-1" /> {tree.path}
        </>
    )

export const ComponentTitleWithIconAndKind: React.FunctionComponent<{
    component: {
        __typename: 'Component'
        name: string
        kind: ComponentKind
    }
    strong?: boolean
}> = ({ component, strong = true }) => (
    <>
        <CatalogComponentIcon component={component} className="icon-inline mr-1" />
        <span className={strong ? 'font-weight-bold' : undefined}>{component.name}</span>
        <span className="text-muted ml-1">{component.kind.toLowerCase()}</span>
    </>
)