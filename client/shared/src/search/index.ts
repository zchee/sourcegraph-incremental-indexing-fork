import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { SearchPatternType } from '@sourcegraph/shared/src/graphql-operations'
import { ISavedSearch } from '@sourcegraph/shared/src/graphql/schema'
import { AggregateStreamingSearchResults, StreamSearchOptions } from '@sourcegraph/shared/src/search/stream'
import { memoizeObservable } from '@sourcegraph/shared/src/util/memoizeObservable'

import { PlatformContext } from '../platform/context'

import {
    EventLogResult,
    isSearchContextAvailable,
    fetchAutoDefinedSearchContexts,
    fetchSearchContexts,
    fetchSearchContext,
    fetchSearchContextBySpec,
    createSearchContext,
    updateSearchContext,
    deleteSearchContext,
    getUserSearchContextNamespaces,
} from './backend'

export interface CaseSensitivityProps {
    caseSensitive: boolean
    setCaseSensitivity: (caseSensitive: boolean) => void
}

export interface SearchPatternTypeProps {
    patternType: SearchPatternType
}

export interface SearchPatternTypeMutationProps {
    setPatternType: (patternType: SearchPatternType) => void
}

export interface SearchContextProps {
    searchContextsEnabled: boolean
    showSearchContext: boolean
    showSearchContextManagement: boolean
    hasUserAddedRepositories: boolean
    hasUserAddedExternalServices: boolean
    defaultSearchContextSpec: string
    selectedSearchContextSpec?: string
    setSelectedSearchContextSpec: (spec: string) => void
    getUserSearchContextNamespaces: typeof getUserSearchContextNamespaces
    fetchAutoDefinedSearchContexts: typeof fetchAutoDefinedSearchContexts
    fetchSearchContexts: typeof fetchSearchContexts
    isSearchContextSpecAvailable: typeof isSearchContextSpecAvailable
    fetchSearchContext: typeof fetchSearchContext
    fetchSearchContextBySpec: typeof fetchSearchContextBySpec
    createSearchContext: typeof createSearchContext
    updateSearchContext: typeof updateSearchContext
    deleteSearchContext: typeof deleteSearchContext
}

export type SearchContextInputProps = Pick<
    SearchContextProps,
    | 'searchContextsEnabled'
    | 'showSearchContext'
    | 'hasUserAddedRepositories'
    | 'hasUserAddedExternalServices'
    | 'showSearchContextManagement'
    | 'defaultSearchContextSpec'
    | 'selectedSearchContextSpec'
    | 'setSelectedSearchContextSpec'
    | 'fetchAutoDefinedSearchContexts'
    | 'fetchSearchContexts'
    | 'getUserSearchContextNamespaces'
>

export interface HomePanelsProps {
    showEnterpriseHomePanels: boolean
    fetchSavedSearches: () => Observable<ISavedSearch[]>
    fetchRecentSearches: (userId: string, first: number) => Observable<EventLogResult | null>
    fetchRecentFileViews: (userId: string, first: number) => Observable<EventLogResult | null>

    /** Function that returns current time (for stability in visual tests). */
    now?: () => Date
}

export interface SearchStreamingProps {
    streamSearch: (
        queryObservable: Observable<string>,
        options: StreamSearchOptions
    ) => Observable<AggregateStreamingSearchResults>
}

export const isSearchContextSpecAvailable = memoizeObservable(
    ({ spec, platformContext }: { spec: string; platformContext: Pick<PlatformContext, 'requestGraphQL'> }) =>
        isSearchContextAvailable(spec, platformContext),
    ({ spec }) => spec
)

export const getAvailableSearchContextSpecOrDefault = memoizeObservable(
    ({
        spec,
        defaultSpec,
        platformContext,
    }: {
        spec: string
        defaultSpec: string
        platformContext: Pick<PlatformContext, 'requestGraphQL'>
    }) => isSearchContextAvailable(spec, platformContext).pipe(map(isAvailable => (isAvailable ? spec : defaultSpec))),
    ({ spec, defaultSpec }) => `${spec}:${defaultSpec}`
)
