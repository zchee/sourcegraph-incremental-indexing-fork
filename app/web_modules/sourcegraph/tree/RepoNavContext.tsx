// tslint:disable

import * as React from "react";

import Component from "sourcegraph/Component";

import {Link} from "react-router";
import urlTo from "sourcegraph/util/urlTo";
import breadcrumb from "sourcegraph/util/breadcrumb";

import * as styles from "sourcegraph/components/styles/breadcrumb.css";

class RepoNavContext extends Component<any, any> {
	static propTypes = {
		params: React.PropTypes.object.isRequired,
	};

	reconcileState(state, props) {
		Object.assign(state, props);
		state.treePath = Array.isArray(props.params.splat) ? props.params.splat[1] : ""; // on the root of the tree, splat is a string
	}

	render(): JSX.Element | null {
		if (!this.state.treePath) return null;
		let pathParts = this.state.treePath.split("/");
		let pathBreadcrumb = breadcrumb(
			`/${this.state.treePath}`,
			(i) => <span key={i} className={styles.sep}>/</span>,
			(path, component, i, isLast) => (
				<Link to={urlTo("tree", Object.assign({}, this.state.params, {splat: [this.state.params.splat[0], pathParts.slice(0, i).join("/")]}))}
					key={i}
					className={isLast ? styles.active : styles.inactive}>
					{component}
				</Link>
			),
		);

		return (
			<span>{pathBreadcrumb}</span>
		);
	}
}

export default RepoNavContext;
