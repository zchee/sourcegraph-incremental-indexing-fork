import React from 'react'

import classNames from 'classnames'
import CancelIcon from 'mdi-react/CancelIcon'

import { Icon } from '@sourcegraph/wildcard'

import styles from './EmptyPanelView.module.scss'

interface EmptyPanelViewProps {
    className?: string
}

export const EmptyPanelView: React.FunctionComponent<EmptyPanelViewProps> = props => {
    const { className, children } = props

    return (
        <div className={classNames(styles.emptyPanel, className)}>
            {children || (
                <>
                    <Icon className="mr-2" as={CancelIcon} /> Nothing to show here
                </>
            )}
        </div>
    )
}
