import classNames from 'classnames'
import React, { useCallback, useEffect, useState } from 'react'
import { NavLink } from 'react-router-dom'
import { Form } from 'reactstrap'

import { useDebounce } from '@sourcegraph/wildcard'

import { CatalogEntityFiltersProps } from '../../../../core/entity-filters'

interface Props extends CatalogEntityFiltersProps {
    size: 'sm' | 'lg'
    className?: string
}

export const EntityListFilters: React.FunctionComponent<Props> = ({ filters, onFiltersChange, size, className }) => {
    const [query, setQuery] = useState(filters.query)

    const debouncedQuery = useDebounce(query, 200)
    useEffect(() => {
        if (filters.query !== debouncedQuery) {
            onFiltersChange({ ...filters, query: debouncedQuery })
        }
    }, [filters, onFiltersChange, debouncedQuery])

    const onQueryChange = useCallback<React.ChangeEventHandler<HTMLInputElement>>(
        event => setQuery(event.currentTarget.value),
        []
    )

    const onSubmit = useCallback<React.FormEventHandler<HTMLFormElement>>(
        event => {
            event.preventDefault()
            onFiltersChange({ ...filters, query })
        },
        [filters, onFiltersChange, query]
    )

    return (
        <Form className={className} onSubmit={onSubmit}>
            <div className="btn-group" role="group">
                <NavLink to="/catalog" exact={true} className="btn border" activeClassName="btn-primary">
                    List
                </NavLink>
                <NavLink to="/catalog/graph" exact={true} className="btn border" activeClassName="btn-primary">
                    Graph
                </NavLink>
            </div>
            <div className={classNames('form-group mb-0')}>
                <label htmlFor="entity-list-filters__query" className="sr-only">
                    Query
                </label>
                <input
                    id="entity-list-filters__query"
                    className={classNames('form-control')}
                    type="search"
                    onChange={onQueryChange}
                    placeholder="Search..."
                    value={query}
                />
            </div>
            <button type="submit" className="sr-only">
                Filter
            </button>
        </Form>
    )
}