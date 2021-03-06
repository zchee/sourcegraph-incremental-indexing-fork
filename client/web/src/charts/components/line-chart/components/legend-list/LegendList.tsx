import React from 'react'

import classNames from 'classnames'

import { LineChartSeries } from '../../types'
import { getLineColor } from '../../utils'

import styles from './LegendList.module.scss'

interface LegendListProps {
    series: LineChartSeries<any>[]
    className?: string
}

export const LegendList: React.FunctionComponent<LegendListProps> = props => {
    const { series, className } = props

    return (
        <ul className={classNames(styles.legendList, className)}>
            {series.map(line => (
                <li key={line.dataKey.toString()} className={styles.legendItem}>
                    <div
                        /* eslint-disable-next-line react/forbid-dom-props */
                        style={{ backgroundColor: getLineColor(line) }}
                        className={styles.legendMark}
                    />
                    {line.name}
                </li>
            ))}
        </ul>
    )
}
